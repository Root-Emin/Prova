package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
)

// EmailVerificationRepository stores active e-mail-verification challenges.
// The production implementation is intentionally Redis-only; PostgreSQL has
// no adapter for this port.
type EmailVerificationRepository interface {
	// Save replaces the current challenge for the account, invalidating any
	// previously issued OTP in one Redis SET.
	Save(ctx context.Context, challenge *model.EmailVerificationChallenge) error
	// Get returns the current challenge, including an exhausted one, so the
	// resend cooldown cannot be bypassed by spending the attempt budget.
	Get(ctx context.Context, userID uuid.UUID) (*model.EmailVerificationChallenge, error)
	// GetActive returns only a challenge that may still be redeemed.
	GetActive(ctx context.Context, userID uuid.UUID, now time.Time) (*model.EmailVerificationChallenge, error)
	// IncrementAttempts atomically charges one failed guess, provided challengeID
	// still names the current challenge.
	IncrementAttempts(ctx context.Context, userID, challengeID uuid.UUID, now time.Time) (int, error)
	// Consume atomically deletes the current active challenge. false means the
	// caller lost a replay/race or the challenge is no longer usable.
	Consume(ctx context.Context, userID, challengeID uuid.UUID, now time.Time) (bool, error)
	// Delete removes the challenge only if challengeID is still current. It is
	// used to roll back state after a provider delivery failure without deleting
	// a newer concurrent request.
	Delete(ctx context.Context, userID, challengeID uuid.UUID) error
}
