package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/masterfabric-go/masterfabric/internal/application/iam/dto"
	iamEvent "github.com/masterfabric-go/masterfabric/internal/domain/iam/event"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
	"github.com/masterfabric-go/masterfabric/internal/shared/events"
)

type RegisterUseCase struct {
	users  repository.UserRepository
	events events.EventBus
	now    func() time.Time
}

// NewRegisterUseCase provisions an account selected by an administrator.
// Delivery deliberately does not happen here: the first Desktop sign-in is
// the single point that requests, sends, and verifies the login code.
func NewRegisterUseCase(users repository.UserRepository, eventBus events.EventBus) *RegisterUseCase {
	return &RegisterUseCase{users: users, events: eventBus, now: time.Now}
}

func (uc *RegisterUseCase) Execute(ctx context.Context, req dto.RegisterRequest, requestIP string) (*dto.RegisterResponse, error) {
	email := model.NormalizeEmail(req.Email)
	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)
	if !validEmail(email) {
		return nil, domainErr.New(domainErr.ErrValidation, "valid email is required", nil)
	}
	if firstName == "" || len(firstName) > 255 || len(lastName) > 255 {
		return nil, domainErr.New(domainErr.ErrValidation, "first name is required", nil)
	}
	existing, err := uc.users.GetByEmail(ctx, email)
	switch {
	case err == nil:
		// Provisioning a pending invite is idempotent. An accidental second
		// submission must not create a second account or send an unexpected mail.
		return uc.provisionExisting(ctx, existing, firstName, lastName)
	case errors.Is(err, domainErr.ErrNotFound):
		// Fresh invite path below.
	default:
		return nil, err
	}

	user := &model.User{
		Email: email, FirstName: firstName, LastName: lastName,
		Status: model.UserStatusInactive, EmailVerifiedAt: nil,
	}
	if err := uc.users.Create(ctx, user); err != nil {
		return nil, err
	}
	if uc.events != nil {
		_ = uc.events.Publish(ctx, events.TopicIAM, iamEvent.UserRegistered{
			UserID: user.ID, Email: user.Email, Timestamp: uc.now().UTC(),
		})
	}
	return &dto.RegisterResponse{Registered: true, Email: user.Email}, nil
}

// provisionExisting leaves an already-pending account ready for its first
// Desktop login. Verified and deleted accounts keep rejecting so the admin UI
// can surface a real conflict rather than silently changing membership.
func (uc *RegisterUseCase) provisionExisting(ctx context.Context, user *model.User, firstName, lastName string) (*dto.RegisterResponse, error) {
	if user.IsDeleted() || user.IsEmailVerified() {
		return nil, domainErr.New(domainErr.ErrAlreadyExists, "an account already exists for this email", nil)
	}

	updated := false
	if user.FirstName == "" && firstName != "" {
		user.FirstName = firstName
		updated = true
	}
	if user.LastName == "" && lastName != "" {
		user.LastName = lastName
		updated = true
	}
	if updated {
		if err := uc.users.Update(ctx, user); err != nil {
			return nil, err
		}
	}
	return &dto.RegisterResponse{Registered: true, Email: user.Email}, nil
}
