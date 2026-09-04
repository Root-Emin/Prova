package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

// challengeBytes, meydan okuma metninin entropi uzunluğu.
//
// 32 bayt yeterli: challenge'ın tek işi tahmin edilememek ve tekrar
// etmemek. Daha uzun olması imzanın güvenliğini artırmaz, yalnızca
// e-postada ve HTTP gövdesinde taşınan veriyi büyütür.
const challengeBytes = 32

// DeviceSignatureService, cihaz anahtar çiftiyle imza doğrular.
//
// Ed25519 seçildi: sabit ve küçük anahtar/imza boyutu (32/64 bayt), tüm
// büyük platformlarda yerel destek, ve parametre seçimi gerektirmeyen tek
// bir eğri. RSA olsaydı istemcinin anahtar boyutu ve dolgu seçimi doğrulama
// yüzeyine sızardı.
type DeviceSignatureService struct{}

// NewDeviceSignatureService wires the service.
func NewDeviceSignatureService() *DeviceSignatureService { return &DeviceSignatureService{} }

// GenerateChallenge, imzalanacak rastgele metni üretir.
func (s *DeviceSignatureService) GenerateChallenge() (string, error) {
	buf := make([]byte, challengeBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("challenge üretilemedi: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// Verify, challenge'ın verilen açık anahtarla imzalandığını doğrular.
//
// İmza ham challenge METNİ üzerinde doğrulanıyor, çözülmüş baytları üzerinde
// değil: istemcinin ne imzalayacağı belirsiz kalmamalı, ve "gördüğün dizeyi
// imzala" tarif edilmesi en zor olan şeydir.
func (s *DeviceSignatureService) Verify(publicKeyB64, challenge, signatureB64 string) error {
	publicKey, err := decodeKey(publicKeyB64)
	if err != nil {
		return err
	}
	signature, err := decodeSignature(signatureB64)
	if err != nil {
		return err
	}
	if !ed25519.Verify(publicKey, []byte(challenge), signature) {
		return ErrSignatureMismatch
	}
	return nil
}

// ValidatePublicKey, kayıt sırasında gelen anahtarın kullanılabilir olduğunu
// doğrular. Geçersiz bir anahtarı kaydetmek, cihazı bir daha hiç
// doğrulanamaz hâle getirirdi.
func (s *DeviceSignatureService) ValidatePublicKey(publicKeyB64 string) error {
	_, err := decodeKey(publicKeyB64)
	return err
}

// Hata değerleri.
var (
	// ErrSignatureMismatch, imza doğrulamasının başarısız olması.
	ErrSignatureMismatch = errors.New("cihaz imzası doğrulanamadı")
	// ErrInvalidPublicKey, anahtarın Ed25519 açık anahtarı olmaması.
	ErrInvalidPublicKey = errors.New("geçersiz cihaz açık anahtarı")
	// ErrInvalidSignature, imzanın çözülememesi ya da yanlış uzunlukta olması.
	ErrInvalidSignature = errors.New("geçersiz cihaz imzası")
)

// decodeKey, base64 anahtarı çözer. Hem standart hem URL-güvenli alfabe
// kabul ediliyor: hangi kodlamayı kullanacağı istemci platformuna göre
// değişiyor ve bu, doğrulamanın kırılmasına değecek bir ayrım değil.
func decodeKey(encoded string) (ed25519.PublicKey, error) {
	raw, err := decodeBase64(encoded)
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil, ErrInvalidPublicKey
	}
	return ed25519.PublicKey(raw), nil
}

func decodeSignature(encoded string) ([]byte, error) {
	raw, err := decodeBase64(encoded)
	if err != nil || len(raw) != ed25519.SignatureSize {
		return nil, ErrInvalidSignature
	}
	return raw, nil
}

func decodeBase64(encoded string) ([]byte, error) {
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding,
		base64.URLEncoding, base64.RawURLEncoding,
	} {
		if raw, err := enc.DecodeString(encoded); err == nil {
			return raw, nil
		}
	}
	return nil, errors.New("base64 çözülemedi")
}
