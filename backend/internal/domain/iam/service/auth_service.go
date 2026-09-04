package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TokenClaims represents the contents of a JWT token.
type TokenClaims struct {
	UserID         uuid.UUID `json:"user_id"`
	Email          string    `json:"email"`
	OrganizationID uuid.UUID `json:"organization_id,omitempty"`
	Roles          []string  `json:"roles,omitempty"`
	Permissions    []string  `json:"permissions,omitempty"`

	// TokenID, JWT'nin jti değeri. Üretimde boş verilir, doğrulamada dolar.
	//
	// İptal, imzalı bir token için ancak bir denylist ile mümkündür: cihaz
	// iptal edildiğinde o cihazın token'ı hâlâ geçerli imzaya sahiptir ve
	// yalnızca kimliğinden tanınabilir.
	TokenID string `json:"-"`
	// DeviceID, token'ın bağlı olduğu cihaz. Cihaz iptalinde o cihaza ait
	// token'ların tamamı reddedilir.
	DeviceID *uuid.UUID `json:"device_id,omitempty"`
	// RefreshFamilyID, token'ı doğuran refresh zincirinin ailesi. Çıkış
	// işleminin aileyi iptal edebilmesi için access token'da taşınır; aksi
	// hâlde çıkış yapan kullanıcının refresh token'ı yaşamaya devam eder.
	RefreshFamilyID *uuid.UUID `json:"refresh_family_id,omitempty"`
	// ExpiresAt, token'ın sona erme anı. Denylist girdisinin ne kadar
	// saklanacağını belirler: token zaten sona erdikten sonra denylist'te
	// tutmanın anlamı yok.
	ExpiresAt time.Time `json:"-"`
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
