package usecase

import (
	"context"
	stdErrors "errors"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/masterfabric-go/masterfabric/internal/application/iam/dto"
	auditService "github.com/masterfabric-go/masterfabric/internal/domain/audit/service"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// PasswordLoginUseCase supports the local/test password credential while the
// mailbox OTP flow remains available for accounts that do not have a hash.
type PasswordLoginUseCase struct {
	VerifyDeps
	now func() time.Time
}

func NewPasswordLoginUseCase(deps VerifyDeps) *PasswordLoginUseCase {
	if deps.Audit == nil {
		deps.Audit = auditService.NoopRecorder{}
	}
	if deps.Log == nil {
		deps.Log = slog.Default()
	}
	return &PasswordLoginUseCase{VerifyDeps: deps, now: time.Now}
}

func (uc *PasswordLoginUseCase) Execute(ctx context.Context, req dto.PasswordLoginRequest, requestIP string) (*dto.VerifyLoginCodeResponse, error) {
	started := uc.now()
	complete := &VerifyLoginCodeUseCase{VerifyDeps: uc.VerifyDeps, now: uc.now}
	defer complete.padResponseTime(started)

	email := model.NormalizeEmail(req.Email)
	if email == "" || req.Password == "" {
		return nil, invalidPassword()
	}

	user, err := uc.Users.GetByEmail(ctx, email)
	if err != nil {
		if stdErrors.Is(err, domainErr.ErrNotFound) {
			return nil, invalidPassword()
		}
		return nil, err
	}
	now := uc.now().UTC()

	if user.IsDeleted() || user.PasswordHash == "" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		uc.failedAttempt(ctx, user, requestIP, now)
		return nil, invalidPassword()
	}
	if user.IsLocked(now) {
		uc.Audit.Record(ctx, auditService.Entry{UserID: &user.ID, Action: auditService.ActionLoginLocked, ResourceType: "user", ResourceID: user.ID.String(), IPAddress: requestIP})
		return nil, invalidPassword()
	}
	if !user.IsActive() && !user.IsPendingDeletion() {
		return nil, domainErr.New(domainErr.ErrForbidden, "account is not active", nil)
	}
	if user.FailedAttempts > 0 || user.LockedUntil != nil {
		if err := uc.Users.ClearFailedAttempts(ctx, user.ID); err != nil {
			uc.Log.ErrorContext(ctx, "başarısız deneme sayacı sıfırlanamadı", "user_id", user.ID, "error", err)
		}
	}

	return complete.CompleteLogin(ctx, user, req.Device, req.DeviceSignature, requestIP, now, "password")
}

func invalidPassword() error {
	return domainErr.New(domainErr.ErrUnauthorized, "invalid email or password", nil)
}

func (uc *PasswordLoginUseCase) failedAttempt(ctx context.Context, user *model.User, requestIP string, now time.Time) {
	if uc.TokenCfg.MaxFailedAttempts <= 0 {
		return
	}
	attempts, err := uc.Users.RecordFailedAttempt(ctx, user.ID, uc.TokenCfg.MaxFailedAttempts, now.Add(uc.TokenCfg.LockDuration))
	if err != nil {
		uc.Log.ErrorContext(ctx, "şifre giriş denemesi kaydedilemedi", "user_id", user.ID, "error", err)
		return
	}
	uc.Audit.Record(ctx, auditService.Entry{UserID: &user.ID, Action: auditService.ActionLoginFailed, ResourceType: "user", ResourceID: user.ID.String(), IPAddress: requestIP, Metadata: map[string]any{"reason": "password_mismatch", "attempts": attempts}})
	if attempts >= uc.TokenCfg.MaxFailedAttempts {
		uc.Audit.Record(ctx, auditService.Entry{UserID: &user.ID, Action: auditService.ActionLoginLocked, ResourceType: "user", ResourceID: user.ID.String(), IPAddress: requestIP})
	}
}
