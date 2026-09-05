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
	users        repository.UserRepository
	verification *RequestEmailVerificationCodeUseCase
	events       events.EventBus
	now          func() time.Time
}

func NewRegisterUseCase(users repository.UserRepository, verification *RequestEmailVerificationCodeUseCase, eventBus events.EventBus) *RegisterUseCase {
	return &RegisterUseCase{users: users, verification: verification, events: eventBus, now: time.Now}
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
	if _, err := uc.users.GetByEmail(ctx, email); err == nil {
		return nil, domainErr.New(domainErr.ErrAlreadyExists, "an account already exists for this email", nil)
	} else if !errors.Is(err, domainErr.ErrNotFound) {
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
	if uc.verification == nil {
		return nil, domainErr.New(domainErr.ErrInternal, "email verification is unavailable", nil)
	}
	verification, err := uc.verification.ExecuteForUser(ctx, user, requestIP)
	if err != nil {
		// The persisted account intentionally remains inactive and unverified;
		// the public resend mutation can retry delivery later.
		return nil, err
	}
	return &dto.RegisterResponse{
		Registered: true, Email: user.Email,
		ExpiresInSeconds:   verification.ExpiresInSeconds,
		ResendAfterSeconds: verification.ResendAfterSeconds,
	}, nil
}
