package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"

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
	if len(cfg.CodePepper) < 16 {
		return nil, fmt.Errorf("auth: AUTH_CODE_PEPPER must be at least 16 characters")
	}
	length := cfg.CodeLength
	if length <= 0 {
		length = 6
	}
	return &LoginCodeService{pepper: []byte(cfg.CodePepper), length: length}, nil
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
	if digest == "" || code == "" {
		return false
	}
	return hmac.Equal([]byte(s.digest(code)), []byte(digest))
}

func (s *LoginCodeService) digest(code string) string {
	mac := hmac.New(sha256.New, s.pepper)
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}
