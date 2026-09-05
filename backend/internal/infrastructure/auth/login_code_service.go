package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/shared/config"
)

// LoginCodeService implements service.LoginCodeService with a peppered
// HMAC-SHA256 digest.
//
// A password hash (bcrypt/argon2) would be the wrong tool here. Those buy work
// factor against offline guessing of a high-entropy secret; a six-digit code
// has ~20 bits of entropy, so no work factor saves it once the digest leaks.
// What actually protects it is the pepper — a secret the database does not
// contain — plus a short lifetime and a hard attempt limit. HMAC also keeps
// verification cheap enough that the endpoint is not itself a denial-of-service
// lever.
type LoginCodeService struct {
	pepper []byte
	length int
}

// NewLoginCodeService creates the service from config. An empty pepper is
// rejected: without it the digest column is one rainbow table away from
// plaintext codes.
func NewLoginCodeService(cfg config.AuthConfig) (*LoginCodeService, error) {
	if cfg.CodeLength != 6 {
		return nil, fmt.Errorf("auth: AUTH_CODE_LENGTH must be exactly 6")
	}
	return newSixDigitCodeService(cfg.CodePepper, "AUTH_CODE_PEPPER")
}

// NewEmailVerificationCodeService reuses the same cryptographic primitive
// under a dedicated pepper/configuration. The business workflows remain
// separate even though secure random generation and HMAC are shared.
func NewEmailVerificationCodeService(cfg config.EmailVerificationConfig) (*LoginCodeService, error) {
	return newSixDigitCodeService(cfg.Pepper, "EMAIL_VERIFICATION_OTP_PEPPER")
}

func newSixDigitCodeService(pepper, setting string) (*LoginCodeService, error) {
	if len(pepper) < 16 {
		return nil, fmt.Errorf("auth: %s must be at least 16 characters", setting)
	}
	return &LoginCodeService{pepper: []byte(pepper), length: 6}, nil
}

// Generate returns a uniformly random numeric code and its digest.
func (s *LoginCodeService) Generate() (string, string, error) {
	// Rejection-free uniform sampling over the whole numeric range, so every
	// code including leading zeros is equally likely. Sampling digit by digit
	// with modulo would bias the distribution and shrink the effective space.
	upper := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(s.length)), nil)
	n, err := rand.Int(rand.Reader, upper)
	if err != nil {
		return "", "", fmt.Errorf("auth: generate login code: %w", err)
	}

	code := fmt.Sprintf("%0*d", s.length, n)
	return code, s.digest(code), nil
}

// Matches implements service.LoginCodeService.
func (s *LoginCodeService) Matches(digest, code string) bool {
	if digest == "" || !s.validCode(code) {
		return false
	}
	return hmac.Equal([]byte(s.digest(code)), []byte(digest))
}

// GenerateFor returns a fresh code whose digest is bound to one account and
// its current normalized address. The plaintext is returned only to the mail
// adapter; neither this service nor its callers persist it.
func (s *LoginCodeService) GenerateFor(userID uuid.UUID, normalizedEmail string) (string, string, error) {
	code, _, err := s.Generate()
	if err != nil {
		return "", "", err
	}
	return code, s.boundDigest(userID, normalizedEmail, code), nil
}

// MatchesFor compares a digest in constant time after binding the candidate
// code to the immutable user ID and normalized current e-mail.
func (s *LoginCodeService) MatchesFor(userID uuid.UUID, normalizedEmail, digest, code string) bool {
	if digest == "" || !s.validCode(code) || userID == uuid.Nil || strings.TrimSpace(normalizedEmail) == "" {
		return false
	}
	return hmac.Equal([]byte(s.boundDigest(userID, normalizedEmail, code)), []byte(digest))
}

func (s *LoginCodeService) digest(code string) string {
	mac := hmac.New(sha256.New, s.pepper)
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *LoginCodeService) boundDigest(userID uuid.UUID, normalizedEmail, code string) string {
	mac := hmac.New(sha256.New, s.pepper)
	_, _ = mac.Write([]byte(userID.String() + ":" + strings.ToLower(strings.TrimSpace(normalizedEmail)) + ":" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *LoginCodeService) validCode(code string) bool {
	if len(code) != s.length {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
