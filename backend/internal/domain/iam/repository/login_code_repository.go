package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
)

// LoginCodeRepository persists one-time login codes.
//
// All methods take an already-normalized address (see model.NormalizeEmail).
type LoginCodeRepository interface {
	// Create stores a newly issued code.
	Create(ctx context.Context, code *model.LoginCode) error

	// GetLatestActive returns the most recent code for the address that has not
	// been consumed, expired or exhausted, or ErrNotFound when there is none.
	GetLatestActive(ctx context.Context, email string, purpose model.LoginCodePurpose, now time.Time) (*model.LoginCode, error)

	// GetLatest returns the most recent code for the address regardless of its
	// state. The request path needs this to enforce the resend cooldown, which
	// must apply even to a code that was already used.
	GetLatest(ctx context.Context, email string, purpose model.LoginCodePurpose) (*model.LoginCode, error)

	// IncrementAttempts records a failed guess and returns the new count.
	IncrementAttempts(ctx context.Context, id uuid.UUID) (int, error)

	// MarkConsumed burns the code so it cannot be redeemed twice.
	MarkConsumed(ctx context.Context, id uuid.UUID, at time.Time) error

	// InvalidateActive consumes every outstanding code for the address. Issuing
	// a new code must retire the previous one, otherwise several valid codes
	// widen the guessing surface for the whole TTL.
	InvalidateActive(ctx context.Context, email string, purpose model.LoginCodePurpose, at time.Time) error

	// CountSince counts codes issued to the address after the given instant,
	// backing the per-address rate limit.
	CountSince(ctx context.Context, email string, purpose model.LoginCodePurpose, since time.Time) (int, error)

	// DeleteExpiredBefore removes spent rows. Codes are short-lived credentials
	// and are not audit records; keeping them is only risk.
	DeleteExpiredBefore(ctx context.Context, cutoff time.Time) (int64, error)

	// DeleteByEmail, kalıcı silmede adrese ait kod satırlarını düşürür.
	// Bir giriş kodu satırı bir adresi bir zaman damgasına bağlar ve
	// kişisel veridir.
	DeleteByEmail(ctx context.Context, email string) error
}
