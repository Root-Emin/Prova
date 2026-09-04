package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/masterfabric-go/masterfabric/internal/application/iam/dto"
	iamEvent "github.com/masterfabric-go/masterfabric/internal/domain/iam/event"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/service"
	notifyModel "github.com/masterfabric-go/masterfabric/internal/domain/notification/model"
	notify "github.com/masterfabric-go/masterfabric/internal/domain/notification/service"
	"github.com/masterfabric-go/masterfabric/internal/domain/notification/template"
	"github.com/masterfabric-go/masterfabric/internal/shared/config"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
	"github.com/masterfabric-go/masterfabric/internal/shared/events"
	"github.com/masterfabric-go/masterfabric/internal/shared/ratelimit"
)

// newDeviceMailTimeout bounds the background send of the new-device notice.
const newDeviceMailTimeout = 15 * time.Second

// VerifyLoginCodeUseCase redeems a one-time code for a session and pairs the
// device the sign-in came from.
//
// Every failure below — no code on file, expired, exhausted, wrong digits —
// returns the same error. Distinguishing them would tell an attacker which of
// their assumptions was the wrong one, and none of the distinctions help a
// legitimate user, whose next step is "request a new code" in all four cases.
type VerifyLoginCodeUseCase struct {
	users    repository.UserRepository
	codes    repository.LoginCodeRepository
	devices  repository.DeviceRepository
	codeSvc  service.LoginCodeService
	auth     service.AuthService
	sender   notify.Sender
	limiter  ratelimit.Limiter
	eventBus events.EventBus
	cfg      config.AuthConfig
	jwtCfg   config.JWTConfig
	log      *slog.Logger
	sleep    func(time.Duration)
	now      func() time.Time
}

// NewVerifyLoginCodeUseCase wires the use case.
func NewVerifyLoginCodeUseCase(
	users repository.UserRepository,
	codes repository.LoginCodeRepository,
	devices repository.DeviceRepository,
	codeSvc service.LoginCodeService,
	auth service.AuthService,
	sender notify.Sender,
	limiter ratelimit.Limiter,
	eventBus events.EventBus,
	cfg config.AuthConfig,
	jwtCfg config.JWTConfig,
	log *slog.Logger,
) *VerifyLoginCodeUseCase {
	return &VerifyLoginCodeUseCase{
		users:    users,
		codes:    codes,
		devices:  devices,
		codeSvc:  codeSvc,
		auth:     auth,
		sender:   sender,
		limiter:  limiter,
		eventBus: eventBus,
		cfg:      cfg,
		jwtCfg:   jwtCfg,
		log:      log,
		sleep:    time.Sleep,
		now:      time.Now,
	}
}

// errInvalidCode is the single answer every rejection collapses into.
func errInvalidCode() error {
	return domainErr.New(domainErr.ErrUnauthorized, "invalid or expired code", nil)
}

// Execute redeems the code and returns a session token.
func (uc *VerifyLoginCodeUseCase) Execute(ctx context.Context, req dto.VerifyLoginCodeRequest, requestIP string) (*dto.VerifyLoginCodeResponse, error) {
	started := uc.now()
	defer uc.padResponseTime(started)

	email := model.NormalizeEmail(req.Email)
	if email == "" || req.Code == "" {
		return nil, errInvalidCode()
	}

	// The per-code attempt budget stops guessing at one address. This limit
	// stops an attacker who spreads guesses across many addresses, where no
	// single budget is ever exhausted.
	if err := uc.checkIPLimit(ctx, requestIP); err != nil {
		return nil, err
	}

	now := uc.now().UTC()

	record, err := uc.codes.GetLatestActive(ctx, email, model.LoginCodePurposeLogin, now)
	if err != nil {
		if errors.Is(err, domainErr.ErrNotFound) {
			return nil, errInvalidCode()
		}
		return nil, err
	}

	if !uc.codeSvc.Matches(record.CodeDigest, req.Code) {
		uc.recordFailedAttempt(ctx, record, now)
		return nil, errInvalidCode()
	}

	// Burn the code before doing anything else. Two requests arriving with the
	// same correct code must not both produce a session.
	if err := uc.codes.MarkConsumed(ctx, record.ID, now); err != nil {
		return nil, err
	}

	user, err := uc.resolveUser(ctx, email, now)
	if err != nil {
		return nil, err
	}
	if !user.IsActive() {
		return nil, domainErr.New(domainErr.ErrForbidden, "account is not active", nil)
	}

	paired, err := uc.pairDevice(ctx, user, req.Device, now)
	if err != nil {
		return nil, err
	}

	token, err := uc.auth.GenerateToken(ctx, service.TokenClaims{
		UserID: user.ID,
		Email:  user.Email,
	})
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to generate token", err)
	}

	uc.log.InfoContext(ctx, "passwordless login completed",
		"user_id", user.ID,
		"device_paired", paired != nil,
		"device_new", paired != nil && paired.IsNew,
	)

	return &dto.VerifyLoginCodeResponse{
		Token:     token,
		ExpiresAt: now.Add(time.Duration(uc.jwtCfg.ExpirationHours) * time.Hour),
		User: dto.UserInfo{
			ID:        user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Status:    string(user.Status),
			CreatedAt: user.CreatedAt,
		},
		Device: paired,
	}, nil
}

// recordFailedAttempt charges the guess against the code's budget and burns the
// code once the budget is gone, so an exhausted code cannot be revived by
// waiting inside its TTL.
func (uc *VerifyLoginCodeUseCase) recordFailedAttempt(ctx context.Context, record *model.LoginCode, now time.Time) {
	attempts, err := uc.codes.IncrementAttempts(ctx, record.ID)
	if err != nil {
		uc.log.ErrorContext(ctx, "failed to record login code attempt", "code_id", record.ID, "error", err)
		return
	}
	if record.MaxAttempts > 0 && attempts >= record.MaxAttempts {
		if err := uc.codes.MarkConsumed(ctx, record.ID, now); err != nil {
			uc.log.ErrorContext(ctx, "failed to burn exhausted login code", "code_id", record.ID, "error", err)
		}
	}
}

// resolveUser returns the account for the address, creating it when the
// deployment allows self-signup. Provisioning happens here rather than at
// request time so that unredeemed codes leave no account behind.
func (uc *VerifyLoginCodeUseCase) resolveUser(ctx context.Context, email string, now time.Time) (*model.User, error) {
	user, err := uc.users.GetByEmail(ctx, email)
	switch {
	case err == nil:
		if user.EmailVerifiedAt == nil {
			verifiedAt := now
			user.EmailVerifiedAt = &verifiedAt
			if err := uc.users.Update(ctx, user); err != nil {
				return nil, err
			}
		}
		return user, nil

	case errors.Is(err, domainErr.ErrNotFound):
		if !uc.cfg.SelfSignup {
			// The code was valid, so this is not an attacker probing; it is an
			// address that lost its invitation between request and redemption.
			return nil, errInvalidCode()
		}
		verifiedAt := now
		user = &model.User{
			Email:           email,
			Status:          model.UserStatusActive,
			EmailVerifiedAt: &verifiedAt,
		}
		if err := uc.users.Create(ctx, user); err != nil {
			return nil, err
		}
		_ = uc.eventBus.Publish(ctx, events.TopicIAM, iamEvent.UserRegistered{
			UserID:    user.ID,
			Email:     user.Email,
			Timestamp: now,
		})
		return user, nil

	default:
		return nil, err
	}
}

// pairDevice binds the calling machine to the account.
//
// Pairing happens at this exact moment by design: the mailbox has just been
// proven, and the machine is present. Doing it later, as a separate opt-in
// step, would leave sessions that no certificate can be traced back to.
func (uc *VerifyLoginCodeUseCase) pairDevice(ctx context.Context, user *model.User, info *dto.DeviceInfo, now time.Time) (*dto.PairedDevice, error) {
	if info == nil || info.Fingerprint == "" {
		// The web panel has no hardware identity. It signs in fine; it just
		// cannot host a certificate-bearing exam.
		return nil, nil
	}

	device := &model.Device{
		UserID:      user.ID,
		Fingerprint: info.Fingerprint,
		Name:        info.Name,
		Platform:    model.NormalizePlatform(info.Platform),
		LastSeenAt:  now,
	}

	created, err := uc.devices.Pair(ctx, device)
	if err != nil {
		return nil, err
	}
	if device.IsRevoked() {
		return nil, domainErr.New(domainErr.ErrForbidden, "this device has been revoked", nil)
	}

	if created {
		uc.notifyNewDevice(user.Email, device, now)
	}

	return &dto.PairedDevice{
		ID:       device.ID,
		Name:     device.Name,
		Platform: string(device.Platform),
		IsNew:    created,
	}, nil
}

// notifyNewDevice mails the account owner that a machine was paired.
//
// Sent off the request path on purpose. It is the user's warning that someone
// else may hold their mailbox, but a provider stall must not turn a successful
// sign-in into a timeout.
func (uc *VerifyLoginCodeUseCase) notifyNewDevice(email string, device *model.Device, now time.Time) {
	name := device.Name
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), newDeviceMailTimeout)
		defer cancel()

		msg := template.NewDevice(notifyModel.Address{Email: email}, name, now)
		if _, err := uc.sender.Send(ctx, msg); err != nil {
			uc.log.ErrorContext(ctx, "new device notification failed",
				"provider", uc.sender.Name(),
				"device_id", device.ID,
				"error", err,
			)
		}
	}()
}

func (uc *VerifyLoginCodeUseCase) checkIPLimit(ctx context.Context, requestIP string) error {
	if requestIP == "" || uc.limiter == nil || uc.cfg.MaxRequestsPerIP <= 0 {
		return nil
	}
	allowed, retryAfter, err := uc.limiter.Allow(ctx, "verify:ip:"+requestIP, uc.cfg.MaxRequestsPerIP, uc.cfg.RateLimitWindow)
	if err != nil {
		uc.log.WarnContext(ctx, "rate limiter unavailable, allowing verification", "error", err)
		return nil
	}
	if !allowed {
		return domainErr.New(domainErr.ErrRateLimited,
			"too many verification attempts, retry in "+retryAfter.Round(time.Second).String(), nil)
	}
	return nil
}

// padResponseTime keeps a rejection from being faster than an acceptance.
func (uc *VerifyLoginCodeUseCase) padResponseTime(started time.Time) {
	if uc.cfg.MinResponseTime <= 0 {
		return
	}
	if remaining := uc.cfg.MinResponseTime - uc.now().Sub(started); remaining > 0 {
		uc.sleep(remaining)
	}
}
