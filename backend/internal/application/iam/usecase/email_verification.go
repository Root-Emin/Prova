package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

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

type RequestEmailVerificationDeps struct {
	Users      repository.UserRepository
	Challenges repository.EmailVerificationRepository
	Codes      service.AccountBoundLoginCodeService
	Sender     notify.Sender
	Limiter    ratelimit.Limiter
	Cfg        config.EmailVerificationConfig
	Log        *slog.Logger
}

type RequestEmailVerificationCodeUseCase struct {
	RequestEmailVerificationDeps
	now func() time.Time
}

func NewRequestEmailVerificationCodeUseCase(deps RequestEmailVerificationDeps) *RequestEmailVerificationCodeUseCase {
	return &RequestEmailVerificationCodeUseCase{RequestEmailVerificationDeps: deps, now: time.Now}
}

func (uc *RequestEmailVerificationCodeUseCase) Execute(ctx context.Context, req dto.RequestEmailVerificationCodeRequest, requestIP string) (*dto.RequestEmailVerificationCodeResponse, error) {
	email := model.NormalizeEmail(req.Email)
	if !validEmail(email) {
		return nil, domainErr.New(domainErr.ErrValidation, "valid email is required", nil)
	}
	if err := uc.checkSendLimits(ctx, email, requestIP); err != nil {
		return nil, err
	}

	user, err := uc.Users.GetByEmail(ctx, email)
	if errors.Is(err, domainErr.ErrNotFound) {
		// The public resend endpoint must not become an account-enumeration
		// oracle. Unknown and already-verified addresses get the same body.
		return uc.response(), nil
	}
	if err != nil {
		return nil, err
	}
	if user.IsEmailVerified() || user.IsDeleted() {
		return uc.response(), nil
	}
	return uc.requestForUser(ctx, user)
}

// ExecuteForUser is used immediately after registration, where the account is
// already known. Delivery/storage errors are returned so registration cannot
// pretend a usable code was sent.
func (uc *RequestEmailVerificationCodeUseCase) ExecuteForUser(ctx context.Context, user *model.User, requestIP string) (*dto.RequestEmailVerificationCodeResponse, error) {
	if user == nil || user.ID == uuid.Nil {
		return nil, domainErr.New(domainErr.ErrInternal, "email verification requires a persisted user", nil)
	}
	email := model.NormalizeEmail(user.Email)
	if !validEmail(email) {
		return nil, domainErr.New(domainErr.ErrValidation, "valid email is required", nil)
	}
	if err := uc.checkSendLimits(ctx, email, requestIP); err != nil {
		return nil, err
	}
	if user.IsEmailVerified() {
		return nil, domainErr.New(domainErr.ErrConflict, "email is already verified", nil)
	}
	return uc.requestForUser(ctx, user)
}

func (uc *RequestEmailVerificationCodeUseCase) requestForUser(ctx context.Context, user *model.User) (*dto.RequestEmailVerificationCodeResponse, error) {
	if uc.Challenges == nil {
		return nil, domainErr.New(domainErr.ErrInternal, "email verification is unavailable", nil)
	}
	now := uc.now().UTC()
	email := model.NormalizeEmail(user.Email)
	if err := uc.checkCooldown(ctx, user.ID, email, now); err != nil {
		return nil, err
	}

	code, digest, err := uc.Codes.GenerateFor(user.ID, email)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to generate email verification code", err)
	}
	challenge := &model.EmailVerificationChallenge{
		ID:          uuid.New(),
		UserID:      user.ID,
		Email:       email,
		CodeDigest:  digest,
		MaxAttempts: uc.Cfg.MaxAttempts,
		ExpiresAt:   now.Add(uc.Cfg.OTPTTL),
		CreatedAt:   now,
	}
	// One key per immutable account makes this SET the invalidation of every
	// previously issued verification OTP for that account.
	if err := uc.Challenges.Save(ctx, challenge); err != nil {
		return nil, err
	}

	msg := template.EmailVerification(notifyModel.Address{Email: email}, code, uc.Cfg.OTPTTL)
	messageID, err := uc.Sender.Send(ctx, msg)
	if err != nil {
		// A failed delivery must not leave an active challenge that the user
		// could never know. Delete is ID-conditional so a newer request wins.
		if cleanupErr := uc.Challenges.Delete(ctx, user.ID, challenge.ID); cleanupErr != nil {
			uc.Log.ErrorContext(ctx, "failed to clean up undelivered email verification challenge",
				"user_id", user.ID, "challenge_id", challenge.ID, "error", cleanupErr)
		}
		uc.Log.ErrorContext(ctx, "email verification delivery failed",
			"provider", uc.Sender.Name(), "user_id", user.ID, "error", err)
		return nil, domainErr.New(domainErr.ErrInternal, "failed to send email verification code", err)
	}

	uc.Log.InfoContext(ctx, "email verification code sent",
		"provider", uc.Sender.Name(), "message_id", messageID,
		"user_id", user.ID, "challenge_id", challenge.ID)
	return uc.response(), nil
}

func (uc *RequestEmailVerificationCodeUseCase) checkCooldown(ctx context.Context, userID uuid.UUID, email string, now time.Time) error {
	challenge, err := uc.Challenges.Get(ctx, userID)
	if errors.Is(err, domainErr.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err // Redis failures fail closed.
	}
	// A challenge for an old address is invalid and cannot throttle the newly
	// selected address. Save below overwrites it.
	if model.NormalizeEmail(challenge.Email) != email {
		return nil
	}
	if elapsed := now.Sub(challenge.CreatedAt); elapsed < uc.Cfg.ResendCooldown {
		wait := (uc.Cfg.ResendCooldown - elapsed).Round(time.Second)
		return domainErr.New(domainErr.ErrRateLimited, "a verification code was already sent, retry in "+wait.String(), nil)
	}
	return nil
}

func (uc *RequestEmailVerificationCodeUseCase) checkSendLimits(ctx context.Context, email, requestIP string) error {
	if uc.Limiter == nil {
		return domainErr.New(domainErr.ErrInternal, "email verification rate limiter is unavailable", nil)
	}
	keys := []struct {
		key     string
		limit   int
		enabled bool
	}{
		{key: "send:email:" + emailRateKey(email), limit: uc.Cfg.MaxRequestsPerEmail, enabled: true},
		{key: "send:ip:" + requestIP, limit: uc.Cfg.MaxRequestsPerIP, enabled: requestIP != ""},
	}
	for _, item := range keys {
		if !item.enabled || item.limit <= 0 {
			continue
		}
		allowed, retryAfter, err := uc.Limiter.Allow(ctx, item.key, item.limit, uc.Cfg.RateLimitWindow)
		if err != nil {
			return domainErr.New(domainErr.ErrInternal, "email verification rate limiter is unavailable", err)
		}
		if !allowed {
			return domainErr.New(domainErr.ErrRateLimited, "too many verification code requests, retry in "+retryAfter.Round(time.Second).String(), nil)
		}
	}
	return nil
}

func (uc *RequestEmailVerificationCodeUseCase) response() *dto.RequestEmailVerificationCodeResponse {
	return &dto.RequestEmailVerificationCodeResponse{
		Sent: true, ExpiresInSeconds: int(uc.Cfg.OTPTTL.Seconds()),
		ResendAfterSeconds: int(uc.Cfg.ResendCooldown.Seconds()),
	}
}

type VerifyEmailDeps struct {
	Users      repository.UserRepository
	Challenges repository.EmailVerificationRepository
	Codes      service.AccountBoundLoginCodeService
	Cfg        config.EmailVerificationConfig
	Log        *slog.Logger
}

type VerifyEmailUseCase struct {
	VerifyEmailDeps
	now func() time.Time
}

func NewVerifyEmailUseCase(deps VerifyEmailDeps) *VerifyEmailUseCase {
	return &VerifyEmailUseCase{VerifyEmailDeps: deps, now: time.Now}
}

func invalidEmailVerificationCode() error {
	return domainErr.New(domainErr.ErrUnauthorized, "invalid or expired email verification code", nil)
}

func (uc *VerifyEmailUseCase) Execute(ctx context.Context, req dto.VerifyEmailRequest) (*dto.VerifyEmailResponse, error) {
	email := model.NormalizeEmail(req.Email)
	if !validEmail(email) || !sixDigits(req.Code) {
		return nil, invalidEmailVerificationCode()
	}
	if uc.Challenges == nil {
		return nil, domainErr.New(domainErr.ErrInternal, "email verification is unavailable", nil)
	}
	user, err := uc.Users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domainErr.ErrNotFound) {
			return nil, invalidEmailVerificationCode()
		}
		return nil, err
	}
	now := uc.now().UTC()
	challenge, err := uc.Challenges.GetActive(ctx, user.ID, now)
	if err != nil {
		if errors.Is(err, domainErr.ErrNotFound) {
			return nil, invalidEmailVerificationCode()
		}
		return nil, err // Redis failures fail closed.
	}
	currentEmail := model.NormalizeEmail(user.Email)
	if challenge.UserID != user.ID || challenge.Email != currentEmail || currentEmail != email {
		return nil, invalidEmailVerificationCode()
	}
	if !uc.Codes.MatchesFor(user.ID, currentEmail, challenge.CodeDigest, req.Code) {
		if _, err := uc.Challenges.IncrementAttempts(ctx, user.ID, challenge.ID, now); err != nil && !errors.Is(err, domainErr.ErrNotFound) {
			return nil, err
		}
		return nil, invalidEmailVerificationCode()
	}

	consumed, err := uc.Challenges.Consume(ctx, user.ID, challenge.ID, now)
	if err != nil {
		return nil, err
	}
	if !consumed {
		return nil, invalidEmailVerificationCode()
	}
	updated, err := uc.Users.MarkEmailVerified(ctx, user.ID, currentEmail, now)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, invalidEmailVerificationCode()
	}
	uc.Log.InfoContext(ctx, "email verified", "user_id", user.ID, "challenge_id", challenge.ID)
	return &dto.VerifyEmailResponse{Verified: true, VerifiedAt: now}, nil
}

func sixDigits(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func validEmail(email string) bool {
	if email == "" || strings.ContainsAny(email, "\r\n") {
		return false
	}
	parsed, err := mail.ParseAddress(email)
	return err == nil && model.NormalizeEmail(parsed.Address) == email
}

func emailRateKey(email string) string {
	sum := sha256.Sum256([]byte(model.NormalizeEmail(email)))
	return hex.EncodeToString(sum[:])
}
