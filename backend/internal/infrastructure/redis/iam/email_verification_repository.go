package iam

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	iamRepo "github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

const emailVerificationPrefix = "prova:iam:email-verification:user:"

// EmailVerificationRepo is the sole production persistence adapter for active
// e-mail-verification challenges. Redis TTL is the challenge lifetime; there
// is deliberately no PostgreSQL implementation or fallback.
type EmailVerificationRepo struct {
	redis *redis.Client
}

func NewEmailVerificationRepo(client *redis.Client) *EmailVerificationRepo {
	return &EmailVerificationRepo{redis: client}
}

type storedEmailVerification struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	CodeDigest  string    `json:"code_digest"`
	Attempts    int       `json:"attempts"`
	MaxAttempts int       `json:"max_attempts"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func (r *EmailVerificationRepo) Save(ctx context.Context, challenge *model.EmailVerificationChallenge) error {
	if challenge.UserID == uuid.Nil || challenge.CodeDigest == "" {
		return domainErr.New(domainErr.ErrInternal, "email verification challenge is incomplete", nil)
	}
	if challenge.ID == uuid.Nil {
		challenge.ID = uuid.New()
	}
	if challenge.CreatedAt.IsZero() {
		challenge.CreatedAt = time.Now().UTC()
	}
	ttl := time.Until(challenge.ExpiresAt)
	if ttl <= 0 {
		return domainErr.New(domainErr.ErrInternal, "email verification challenge expired before storage", nil)
	}
	payload, err := json.Marshal(storedEmailVerificationFromModel(challenge))
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to encode email verification challenge", err)
	}
	if err := r.redis.Set(ctx, emailVerificationKey(challenge.UserID), payload, ttl).Err(); err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to store email verification challenge", err)
	}
	return nil
}

func (r *EmailVerificationRepo) Get(ctx context.Context, userID uuid.UUID) (*model.EmailVerificationChallenge, error) {
	raw, err := r.redis.Get(ctx, emailVerificationKey(userID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, emailVerificationNotFound()
	}
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to load email verification challenge", err)
	}
	var stored storedEmailVerification
	if err := json.Unmarshal(raw, &stored); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to decode email verification challenge", err)
	}
	return stored.toModel(), nil
}

func (r *EmailVerificationRepo) GetActive(ctx context.Context, userID uuid.UUID, now time.Time) (*model.EmailVerificationChallenge, error) {
	challenge, err := r.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if challenge.UserID != userID || !challenge.IsActive(now) {
		return nil, emailVerificationNotFound()
	}
	return challenge, nil
}

func (r *EmailVerificationRepo) IncrementAttempts(ctx context.Context, userID, challengeID uuid.UUID, now time.Time) (int, error) {
	key := emailVerificationKey(userID)
	attempts := 0
	err := r.redis.Watch(ctx, func(tx *redis.Tx) error {
		stored, err := getStoredEmailVerification(ctx, tx, key)
		if err != nil {
			return err
		}
		if stored.ID != challengeID || stored.UserID != userID || !now.Before(stored.ExpiresAt) ||
			(stored.MaxAttempts > 0 && stored.Attempts >= stored.MaxAttempts) {
			return emailVerificationNotFound()
		}
		stored.Attempts++
		attempts = stored.Attempts
		return setStoredEmailVerification(ctx, tx, key, stored)
	}, key)
	if err != nil {
		if errors.Is(err, domainErr.ErrNotFound) {
			return 0, err
		}
		return 0, domainErr.New(domainErr.ErrInternal, "failed to increment email verification attempts", err)
	}
	return attempts, nil
}

func (r *EmailVerificationRepo) Consume(ctx context.Context, userID, challengeID uuid.UUID, now time.Time) (bool, error) {
	key := emailVerificationKey(userID)
	consumed := false
	err := r.redis.Watch(ctx, func(tx *redis.Tx) error {
		stored, err := getStoredEmailVerification(ctx, tx, key)
		if err != nil {
			return err
		}
		if stored.ID != challengeID || stored.UserID != userID || !now.Before(stored.ExpiresAt) ||
			(stored.MaxAttempts > 0 && stored.Attempts >= stored.MaxAttempts) {
			return nil
		}
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Del(ctx, key)
			return nil
		})
		if err == nil {
			consumed = true
		}
		return err
	}, key)
	if errors.Is(err, domainErr.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, domainErr.New(domainErr.ErrInternal, "failed to consume email verification challenge", err)
	}
	return consumed, nil
}

func (r *EmailVerificationRepo) Delete(ctx context.Context, userID, challengeID uuid.UUID) error {
	key := emailVerificationKey(userID)
	err := r.redis.Watch(ctx, func(tx *redis.Tx) error {
		stored, err := getStoredEmailVerification(ctx, tx, key)
		if errors.Is(err, domainErr.ErrNotFound) {
			return nil
		}
		if err != nil || stored.ID != challengeID {
			return err
		}
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Del(ctx, key)
			return nil
		})
		return err
	}, key)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to delete email verification challenge", err)
	}
	return nil
}

func getStoredEmailVerification(ctx context.Context, tx *redis.Tx, key string) (storedEmailVerification, error) {
	raw, err := tx.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return storedEmailVerification{}, emailVerificationNotFound()
	}
	if err != nil {
		return storedEmailVerification{}, err
	}
	var stored storedEmailVerification
	if err := json.Unmarshal(raw, &stored); err != nil {
		return storedEmailVerification{}, err
	}
	return stored, nil
}

func setStoredEmailVerification(ctx context.Context, tx *redis.Tx, key string, stored storedEmailVerification) error {
	payload, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	ttl := time.Until(stored.ExpiresAt)
	if ttl <= 0 {
		return emailVerificationNotFound()
	}
	_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, key, payload, ttl)
		return nil
	})
	return err
}

func storedEmailVerificationFromModel(c *model.EmailVerificationChallenge) storedEmailVerification {
	return storedEmailVerification{
		ID: c.ID, UserID: c.UserID, Email: c.Email, CodeDigest: c.CodeDigest,
		Attempts: c.Attempts, MaxAttempts: c.MaxAttempts, ExpiresAt: c.ExpiresAt, CreatedAt: c.CreatedAt,
	}
}

func (c storedEmailVerification) toModel() *model.EmailVerificationChallenge {
	return &model.EmailVerificationChallenge{
		ID: c.ID, UserID: c.UserID, Email: c.Email, CodeDigest: c.CodeDigest,
		Attempts: c.Attempts, MaxAttempts: c.MaxAttempts, ExpiresAt: c.ExpiresAt, CreatedAt: c.CreatedAt,
	}
}

func emailVerificationKey(userID uuid.UUID) string {
	return emailVerificationPrefix + userID.String()
}

func emailVerificationNotFound() error {
	return domainErr.New(domainErr.ErrNotFound, "email verification challenge not found", nil)
}

var _ iamRepo.EmailVerificationRepository = (*EmailVerificationRepo)(nil)
