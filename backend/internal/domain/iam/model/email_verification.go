package model

import (
	"time"

	"github.com/google/uuid"
)

// EmailVerificationChallenge is the short-lived proof required to mark the
// current address on one account as verified. It contains only a peppered
// digest; the six-digit OTP exists only long enough to build the outgoing
// message.
type EmailVerificationChallenge struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	CodeDigest  string    `json:"-"`
	Attempts    int       `json:"attempts"`
	MaxAttempts int       `json:"max_attempts"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func (c *EmailVerificationChallenge) IsExpired(now time.Time) bool {
	return !now.Before(c.ExpiresAt)
}

func (c *EmailVerificationChallenge) IsExhausted() bool {
	return c.MaxAttempts > 0 && c.Attempts >= c.MaxAttempts
}

func (c *EmailVerificationChallenge) IsActive(now time.Time) bool {
	return !c.IsExpired(now) && !c.IsExhausted()
}
