package service

import (
	"context"

	"github.com/google/uuid"
)

// TokenClaims represents the contents of a JWT token.
type TokenClaims struct {
	UserID         uuid.UUID `json:"user_id"`
	Email          string    `json:"email"`
	OrganizationID uuid.UUID `json:"organization_id,omitempty"`
	Roles          []string  `json:"roles,omitempty"`
	Permissions    []string  `json:"permissions,omitempty"`
}

// AuthService defines token operations.
//
// Password hashing and verification used to live here and are gone on purpose.
// Authentication is passwordless: the credential is a one-time code delivered
// to a verified mailbox (see LoginCodeService). Leaving the methods in place
// "just in case" would be an open invitation to reintroduce the surface the
// design removed.
type AuthService interface {
	// GenerateToken creates a JWT token from claims.
	GenerateToken(ctx context.Context, claims TokenClaims) (string, error)
	// ValidateToken validates a JWT token and returns its claims.
	ValidateToken(ctx context.Context, token string) (*TokenClaims, error)
}
