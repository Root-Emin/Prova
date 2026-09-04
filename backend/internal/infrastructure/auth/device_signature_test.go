package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
)

func newKeyPair(t *testing.T) (publicB64 string, private ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("anahtar çifti üretilemedi: %v", err)
	}
	return base64.StdEncoding.EncodeToString(pub), priv
}

func TestDeviceSignature_AcceptsAGenuineSignature(t *testing.T) {
	svc := NewDeviceSignatureService()
	publicKey, privateKey := newKeyPair(t)

	challenge, err := svc.GenerateChallenge()
	if err != nil {
		t.Fatalf("challenge üretilemedi: %v", err)
	}
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(challenge)))

	if err := svc.Verify(publicKey, challenge, signature); err != nil {
		t.Fatalf("gerçek imza doğrulanmalı: %v", err)
	}
}

// Parmak izi kopyalanabilir, imza kopyalanamaz: başka bir anahtarla
// imzalanmış bir challenge kabul edilmemeli.
func TestDeviceSignature_RejectsASignatureFromAnotherKey(t *testing.T) {
	svc := NewDeviceSignatureService()
	publicKey, _ := newKeyPair(t)
	_, attackerKey := newKeyPair(t)

	challenge, _ := svc.GenerateChallenge()
	forged := base64.StdEncoding.EncodeToString(ed25519.Sign(attackerKey, []byte(challenge)))

	if err := svc.Verify(publicKey, challenge, forged); err == nil {
		t.Fatal("başka anahtarla üretilmiş imza reddedilmeli")
	}
}

// Yakalanan bir imza başka bir challenge'a taşınamamalı; challenge'ın tek
// kullanımlık olmasının yanında imzanın metne bağlı olması da gerekiyor.
func TestDeviceSignature_RejectsASignatureForADifferentChallenge(t *testing.T) {
	svc := NewDeviceSignatureService()
	publicKey, privateKey := newKeyPair(t)

	first, _ := svc.GenerateChallenge()
	second, _ := svc.GenerateChallenge()
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(first)))

	if err := svc.Verify(publicKey, second, signature); err == nil {
		t.Fatal("başka bir challenge için üretilmiş imza reddedilmeli")
	}
}

func TestDeviceSignature_ChallengesAreUnique(t *testing.T) {
	svc := NewDeviceSignatureService()

	seen := make(map[string]bool, 50)
	for i := 0; i < 50; i++ {
		challenge, err := svc.GenerateChallenge()
		if err != nil {
			t.Fatalf("challenge üretilemedi: %v", err)
		}
		if seen[challenge] {
			t.Fatal("challenge'lar benzersiz olmalı; tekrar eden bir metin imzayı yeniden oynatılabilir kılar")
		}
		seen[challenge] = true
	}
}

// Hangi base64 alfabesinin kullanılacağı istemci platformuna göre değişiyor
// ve bu, doğrulamanın kırılmasına değecek bir ayrım değil.
func TestDeviceSignature_AcceptsBothBase64Alphabets(t *testing.T) {
	svc := NewDeviceSignatureService()
	pub, priv, _ := ed25519.GenerateKey(nil)
	challenge, _ := svc.GenerateChallenge()
	rawSignature := ed25519.Sign(priv, []byte(challenge))

	encodings := map[string]struct{ key, sig string }{
		"standart":     {base64.StdEncoding.EncodeToString(pub), base64.StdEncoding.EncodeToString(rawSignature)},
		"url-güvenli":  {base64.URLEncoding.EncodeToString(pub), base64.URLEncoding.EncodeToString(rawSignature)},
		"dolgusuz":     {base64.RawStdEncoding.EncodeToString(pub), base64.RawStdEncoding.EncodeToString(rawSignature)},
		"dolgusuz-url": {base64.RawURLEncoding.EncodeToString(pub), base64.RawURLEncoding.EncodeToString(rawSignature)},
	}
	for name, enc := range encodings {
		if err := svc.Verify(enc.key, challenge, enc.sig); err != nil {
			t.Errorf("%s kodlaması kabul edilmeli: %v", name, err)
		}
	}
}

// Geçersiz bir anahtarı kaydetmek, cihazı bir daha hiç doğrulanamaz hâle
// getirir; kayıt anında reddedilmeli.
func TestDeviceSignature_RejectsMalformedPublicKeys(t *testing.T) {
	svc := NewDeviceSignatureService()

	for _, invalid := range []string{"", "not-base64!!", base64.StdEncoding.EncodeToString([]byte("kısa"))} {
		if err := svc.ValidatePublicKey(invalid); err == nil {
			t.Errorf("geçersiz anahtar reddedilmeli: %q", invalid)
		}
	}
}
