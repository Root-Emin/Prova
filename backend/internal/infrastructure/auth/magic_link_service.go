package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// magicLinkTokenBytes, token'ın entropi uzunluğu.
//
// 32 bayt = 256 bit. Bu, tahmin edilmesi pratikte imkânsız olan ve bu yüzden
// pepper gerektirmeyen bir uzay; giriş kodunun 10^6'lık uzayıyla arasındaki
// fark, birinin neden peppered digest gerektirip diğerinin gerektirmediğini
// tek başına açıklıyor.
const magicLinkTokenBytes = 32

// MagicLinkService, bağlantı token'larını üretir ve hash'ler.
type MagicLinkService struct{}

// NewMagicLinkService wires the service.
func NewMagicLinkService() *MagicLinkService { return &MagicLinkService{} }

// Generate, yeni bir token ve onun saklanacak hash'ini üretir.
//
// Token URL-güvenli base64: e-postadaki bağlantıya doğrudan girecek ve
// yüzde-kodlama gerektirmemeli, çünkü kırılan bir bağlantı kullanıcı için
// çalışmayan bir giriş demektir.
func (s *MagicLinkService) Generate() (token, hash string, err error) {
	buf := make([]byte, magicLinkTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("magic link token üretilemedi: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(buf)
	return token, HashMagicToken(token), nil
}

// HashMagicToken, token'ın saklanacak biçimini üretir.
func HashMagicToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Hash implements the token hashing half of the issuer contract.
func (s *MagicLinkService) Hash(token string) string { return HashMagicToken(token) }
