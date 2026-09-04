package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/masterfabric-go/masterfabric/internal/application/iam/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/service"
	notifyModel "github.com/masterfabric-go/masterfabric/internal/domain/notification/model"
	notify "github.com/masterfabric-go/masterfabric/internal/domain/notification/service"
	"github.com/masterfabric-go/masterfabric/internal/domain/notification/template"
	"github.com/masterfabric-go/masterfabric/internal/shared/config"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
	"github.com/masterfabric-go/masterfabric/internal/shared/ratelimit"
)

// RequestLoginCodeUseCase issues a one-time sign-in code and mails it.
//
// The single hardest requirement on this path is that its observable behaviour
// must not depend on whether the address belongs to an account: same status,
// same body, same duration. Three separate mechanisms enforce that — an
// identical response value, a response-time floor, and a rate limit keyed on
// values the caller already knows.
type RequestLoginCodeUseCase struct {
	users   repository.UserRepository
	codes   repository.LoginCodeRepository
	codeSvc service.LoginCodeService
	sender  notify.Sender
	limiter ratelimit.Limiter
	cfg     config.AuthConfig
	log     *slog.Logger
	sleep   func(time.Duration)
	now     func() time.Time
}

// NewRequestLoginCodeUseCase wires the use case.
func NewRequestLoginCodeUseCase(
	users repository.UserRepository,
	codes repository.LoginCodeRepository,
	codeSvc service.LoginCodeService,
	sender notify.Sender,
	limiter ratelimit.Limiter,
	cfg config.AuthConfig,
	log *slog.Logger,
) *RequestLoginCodeUseCase {
	return &RequestLoginCodeUseCase{
		users:   users,
		codes:   codes,
		codeSvc: codeSvc,
		sender:  sender,
		limiter: limiter,
		cfg:     cfg,
		log:     log,
		sleep:   time.Sleep,
		now:     time.Now,
	}
}

// Execute issues a code for the address. requestIP is used for rate limiting
// only and may be empty.
func (uc *RequestLoginCodeUseCase) Execute(ctx context.Context, req dto.RequestLoginCodeRequest, requestIP string) (*dto.RequestLoginCodeResponse, error) {
	started := uc.now()
	defer uc.padResponseTime(started)

	email := model.NormalizeEmail(req.Email)
	if email == "" {
		return nil, domainErr.New(domainErr.ErrValidation, "email is required", nil)
	}

	// IP first: it is the only limit that constrains an attacker sweeping many
	// addresses, and it must apply before any work that touches the database.
	if err := uc.checkLimit(ctx, "ip:"+requestIP, uc.cfg.MaxRequestsPerIP, requestIP != ""); err != nil {
		return nil, err
	}
	if err := uc.checkLimit(ctx, "email:"+email, uc.cfg.MaxRequestsPerEmail, true); err != nil {
		return nil, err
	}

	if err := uc.checkCooldown(ctx, email); err != nil {
		return nil, err
	}

	_, err := uc.users.GetByEmail(ctx, email)
	switch {
	case err == nil:
		// Known address: proceed.
	case errors.Is(err, domainErr.ErrNotFound):
		if !uc.cfg.SelfSignup {
			// Invitation-only deployment: there is nobody to mail. The caller
			// still gets the standard response after the standard delay, so the
			// silence is indistinguishable from a delivered code.
			uc.log.InfoContext(ctx, "login code requested for unknown address", "provisioning", "disabled")
			return uc.response(), nil
		}
		// Self-signup: the account is created when the code is redeemed, not
		// now. Issuing codes to addresses that never verify must not populate
		// the user table.
	default:
		return nil, err
	}

	code, digest, err := uc.codeSvc.Generate()
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to generate login code", err)
	}

	now := uc.now().UTC()

	// Retire outstanding codes before issuing a new one. Several live codes for
	// one address multiply the guessing surface for the whole TTL, and a user
	// who requested a second code has already stopped watching for the first.
	if err := uc.codes.InvalidateActive(ctx, email, model.LoginCodePurposeLogin, now); err != nil {
		return nil, err
	}

	record := &model.LoginCode{
		Email:       email,
		CodeDigest:  digest,
		Purpose:     model.LoginCodePurposeLogin,
		MaxAttempts: uc.cfg.MaxAttempts,
		ExpiresAt:   now.Add(uc.cfg.CodeTTL),
		RequestIP:   requestIP,
		CreatedAt:   now,
	}
	if err := uc.codes.Create(ctx, record); err != nil {
		return nil, err
	}

	msg := template.LoginCode(notifyModel.Address{Email: email}, code, uc.cfg.CodeTTL)
	messageID, err := uc.sender.Send(ctx, msg)
	if err != nil {
		// A code nobody can read is the same as no code at all, so this is a
		// hard failure rather than a silent success. The stored code is left in
		// place: it expires on its own, and burning it here would let a
		// provider hiccup lock the address out of its own retry.
		uc.log.ErrorContext(ctx, "login code delivery failed",
			"provider", uc.sender.Name(),
			"error", err,
		)
		return nil, domainErr.New(domainErr.ErrInternal, "failed to send login code", err)
	}

	// The address is deliberately absent from this line: it is the one field
	// that would turn the log stream into the account list the endpoint refuses
	// to be.
	uc.log.InfoContext(ctx, "login code sent",
		"provider", uc.sender.Name(),
		"message_id", messageID,
		"code_id", record.ID,
	)

	return uc.response(), nil
}

func (uc *RequestLoginCodeUseCase) response() *dto.RequestLoginCodeResponse {
	return &dto.RequestLoginCodeResponse{
		Sent:               true,
		ExpiresInSeconds:   int(uc.cfg.CodeTTL.Seconds()),
		ResendAfterSeconds: int(uc.cfg.ResendCooldown.Seconds()),
	}
}

func (uc *RequestLoginCodeUseCase) checkLimit(ctx context.Context, key string, limit int, enabled bool) error {
	if !enabled || limit <= 0 || uc.limiter == nil {
		return nil
	}
	allowed, retryAfter, err := uc.limiter.Allow(ctx, key, limit, uc.cfg.RateLimitWindow)
	if err != nil {
		// Failing open is the deliberate choice: a Redis outage must not take
		// the only sign-in path down with it. The per-address cooldown below
		// runs off PostgreSQL and still throttles the obvious abuse.
		uc.log.WarnContext(ctx, "rate limiter unavailable, allowing request", "error", err)
		return nil
	}
	if !allowed {
		return domainErr.New(domainErr.ErrRateLimited,
			"too many login code requests, retry in "+retryAfter.Round(time.Second).String(), nil)
	}
	return nil
}

// checkCooldown enforces the minimum gap between two codes for one address.
func (uc *RequestLoginCodeUseCase) checkCooldown(ctx context.Context, email string) error {
	if uc.cfg.ResendCooldown <= 0 {
		return nil
	}

	latest, err := uc.codes.GetLatest(ctx, email, model.LoginCodePurposeLogin)
	if err != nil {
		if errors.Is(err, domainErr.ErrNotFound) {
			return nil
		}
		return err
	}

	elapsed := uc.now().UTC().Sub(latest.CreatedAt)
	if elapsed < uc.cfg.ResendCooldown {
		wait := (uc.cfg.ResendCooldown - elapsed).Round(time.Second)
		return domainErr.New(domainErr.ErrRateLimited,
			"a code was already sent, retry in "+wait.String(), nil)
	}
	return nil
}

// padResponseTime holds the response until the configured floor has elapsed.
//
// Wording alone does not hide account existence. A known address costs a user
// lookup, a code write and a provider round trip; an unknown one costs a
// lookup. That difference is measurable over enough samples, so every answer is
// stretched to the same floor.
func (uc *RequestLoginCodeUseCase) padResponseTime(started time.Time) {
	if uc.cfg.MinResponseTime <= 0 {
		return
	}
	if remaining := uc.cfg.MinResponseTime - uc.now().Sub(started); remaining > 0 {
		uc.sleep(remaining)
	}
}
