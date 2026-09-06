package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	provaModel "github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
)

// Tohum verisindeki sabit kimlikler. cmd/seed bunları yazıyor.
const (
	demoOrgID       = "11111111-1111-4111-8111-111111111111"
	adminEmail      = "yonetici@prova.local"
	traineeEmail    = "calisan@prova.local"
	refundScenario  = "77777777-7777-4777-8777-777777777777"
	kvkkScenario    = "88888888-8888-4888-8888-888888888888"
	calmCharacter   = "44444444-4444-4444-8444-444444444444"
	serviceRubric   = "66666666-6666-4666-8666-666666666666"
	fastProfile     = "99999999-9999-4999-8999-999999999999"
	fastModelName   = provaModel.DefaultPersonaModel
	strongModelName = provaModel.DefaultEvaluatorModel
)

// Geçerli bir TC kimlik numarası (checksum doğru). Yalnızca maskeleme
// kontrolü için; gerçek bir kişiye ait değil.
const sampleTCKN = "10000000146"

// sampleIBAN, maskeleme kontrolü için örnek IBAN.
const sampleIBAN = "TR33 0006 1005 1978 6457 8413 26"

// harness, doğrulama boyunca paylaşılan durum.
type harness struct {
	*client

	// adminToken, tüm izinlere sahip tohum yöneticisi.
	adminToken string
	// traineeToken, yalnızca çalışan izinlerine sahip tohum kullanıcısı.
	traineeToken string
	// runID, bu koşuya özgü sonek. Doğrulama tekrar çalıştırılabilir olmalı
	// ve her koşu kendi kullanıcılarını açmalı.
	runID string
}

func newHarness(c *client) (*harness, error) {
	h := &harness{client: c, runID: randomHex(4)}

	adminToken, err := h.loginWithCode(adminEmail, nil, "")
	if err != nil {
		return nil, fmt.Errorf("yönetici girişi: %w", err)
	}
	h.adminToken = adminToken

	traineeToken, err := h.loginWithCode(traineeEmail, nil, "")
	if err != nil {
		return nil, fmt.Errorf("çalışan girişi: %w", err)
	}
	h.traineeToken = traineeToken

	return h, nil
}

// authResult, giriş yanıtının doğrulamanın ilgilendiği kısmı.
type authResult struct {
	AccessToken    string `json:"accessToken"`
	RefreshToken   string `json:"refreshToken"`
	OrganizationID string `json:"organizationId"`
	User           struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"emailVerified"`
	} `json:"user"`
	Device *struct {
		ID       string `json:"id"`
		Platform string `json:"platform"`
		IsNew    bool   `json:"isNew"`
	} `json:"device"`
}

// requestCode, adrese kod ister.
func (h *harness) requestCode(email string) error {
	resp, err := h.query("", fmt.Sprintf(
		`mutation { requestLoginCode(input:{email:%q}) { sent } }`, email))
	if err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return fmt.Errorf("kod istenemedi: %s", resp.firstError())
	}
	return nil
}

// inviteUser provisions an account through the current administrator flow.
// Existing seeded users are already provisioned and are left unchanged. The
// first Desktop login then verifies the mailbox and activates the membership.
func (h *harness) registerAndVerify(email string) error {
	if email == adminEmail || email == traineeEmail {
		return nil
	}

	resp, err := h.query(h.adminToken, fmt.Sprintf(
		`mutation { inviteUser(input:{email:%q, firstName:"Doğrulama", lastName:"Kullanıcısı", role:EMPLOYEE}) { invited email } }`, email))
	if err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		if strings.Contains(strings.ToLower(resp.firstError()), "already exists") {
			return nil
		}
		return fmt.Errorf("davet başarısız: %s", resp.firstError())
	}
	return nil
}

// deviceInput, giriş isteğine eklenecek cihaz alanı.
type deviceInput struct {
	Fingerprint string
	Name        string
	Platform    string
	PublicKey   string
}

func (d *deviceInput) render() string {
	if d == nil {
		return ""
	}
	parts := []string{fmt.Sprintf("fingerprint:%q", d.Fingerprint)}
	if d.Name != "" {
		parts = append(parts, fmt.Sprintf("name:%q", d.Name))
	}
	if d.Platform != "" {
		parts = append(parts, fmt.Sprintf("platform:%q", d.Platform))
	}
	if d.PublicKey != "" {
		parts = append(parts, fmt.Sprintf("publicKey:%q", d.PublicKey))
	}
	return fmt.Sprintf(", device:{%s}", strings.Join(parts, ", "))
}

// verifyCode, kodu oturuma çevirir ve tam yanıtı döndürür.
func (h *harness) verifyCode(email, code string, device *deviceInput, signature string) (authResult, gqlResponse, error) {
	signatureField := ""
	if signature != "" {
		signatureField = fmt.Sprintf(", deviceSignature:%q", signature)
	}

	resp, err := h.query("", fmt.Sprintf(
		`mutation { verifyLoginCode(input:{email:%q, code:%q%s%s}) {
			accessToken refreshToken organizationId
			user { id email emailVerified }
			device { id platform isNew }
		} }`, email, code, device.render(), signatureField))
	if err != nil {
		return authResult{}, resp, err
	}
	if len(resp.Errors) > 0 {
		return authResult{}, resp, fmt.Errorf("%s", resp.firstError())
	}

	var payload struct {
		VerifyLoginCode authResult `json:"verifyLoginCode"`
	}
	if err := resp.decode(&payload); err != nil {
		return authResult{}, resp, err
	}
	return payload.VerifyLoginCode, resp, nil
}

// loginWithCode, kod akışının tamamını yürütür ve access token döndürür.
func (h *harness) loginWithCode(email string, device *deviceInput, signature string) (string, error) {
	result, err := h.loginFull(email, device, signature)
	if err != nil {
		return "", err
	}
	return result.AccessToken, nil
}

// loginFull, kod akışını yürütür ve tam yanıtı döndürür.
func (h *harness) loginFull(email string, device *deviceInput, signature string) (authResult, error) {
	if err := h.registerAndVerify(email); err != nil {
		return authResult{}, err
	}
	if device == nil && email != adminEmail && email != traineeEmail {
		device = &deviceInput{
			Fingerprint: "fp-" + h.runID + "-" + randomHex(4),
			Name:        "Doğrulama cihazı",
			Platform:    "linux",
		}
	}
	if err := h.requestCode(email); err != nil {
		return authResult{}, err
	}
	body, err := h.mailBody(email)
	if err != nil {
		return authResult{}, err
	}
	code, ok := loginCode(body)
	if !ok {
		return authResult{}, fmt.Errorf("iletide altı haneli kod yok")
	}
	result, _, err := h.verifyCode(email, code, device, signature)
	return result, err
}

// newEmail, bu koşuya özgü bir adres üretir.
func (h *harness) newEmail(prefix string) string {
	return fmt.Sprintf("%s-%s-%s@prova.local", prefix, h.runID, randomHex(3))
}

// joinDemoOrg, kullanıcıyı tohum organizasyonuna alır.
//
// Yeni bir kullanıcı kendi kişisel organizasyonunda açılıyor ve orada
// oynanabilir içerik yok. Doğrulama tohum içeriğini kullandığı için
// kullanıcının demo organizasyonuna taşınması gerekiyor; kişisel üyelik
// siliniyor çünkü çözücü en eski etkin üyeliği seçiyor.
func (h *harness) joinDemoOrg(userID, roleName string) error {
	statements := []string{
		fmt.Sprintf(
			`INSERT INTO organization_users (organization_id, user_id, status) VALUES ('%s','%s','active') ON CONFLICT (organization_id, user_id) DO UPDATE SET status='active'`,
			demoOrgID, userID),
		fmt.Sprintf(
			`DELETE FROM organization_users WHERE user_id='%s' AND organization_id <> '%s'`,
			userID, demoOrgID),
		fmt.Sprintf(
			`INSERT INTO user_roles (user_id, role_id, organization_id) SELECT '%s', id, '%s' FROM roles WHERE scope_id='%s' AND name='%s'`,
			userID, demoOrgID, demoOrgID, roleName),
	}
	for _, sql := range statements {
		if _, err := psql(sql); err != nil {
			return err
		}
	}
	return nil
}

// psql, PostgreSQL'e tek satırlık sorgu çalıştırır.
//
// Doğrulamanın veritabanına doğrudan bakması gerekiyor: "e-posta null oldu
// mu" sorusu API üzerinden sorulamaz, çünkü silinen hesabın API'si zaten
// yok.
func psql(sql string) (string, error) {
	cmd := exec.Command("docker", "exec", "prova-postgres",
		"psql", "-U", "masterfabric", "-d", "masterfabric", "-tAc", sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("psql: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// mongo, MongoDB'ye JavaScript ifadesi çalıştırır.
func mongo(js string) (string, error) {
	cmd := exec.Command("docker", "exec", "prova-mongo",
		"mongosh", "--quiet", "prova", "--eval", js)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("mongosh: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// keyPair, cihaz doğrulaması için Ed25519 anahtar çifti.
type keyPair struct {
	publicB64 string
	private   ed25519.PrivateKey
}

func newKeyPair() (keyPair, error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return keyPair{}, err
	}
	return keyPair{publicB64: base64.StdEncoding.EncodeToString(public), private: private}, nil
}

func (k keyPair) sign(challenge string) string {
	return base64.StdEncoding.EncodeToString(ed25519.Sign(k.private, []byte(challenge)))
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "0000"
	}
	return hex.EncodeToString(buf)
}

// prettyJSON, hata ayıklama çıktısı için.
func prettyJSON(v any) string {
	encoded, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(encoded)
}

// runGo, doğrulama sırasında bir Go komutu çalıştırır.
//
// Yalnızca çalışan sunucudan gözlemlenemeyen kurallar için kullanılıyor
// (üretim sertleştirmesi gibi): geliştirme modunda çalışan bir sunucuda
// üretim davranışını göstermenin başka yolu, ikinci bir sunucu ayağa
// kaldırmak olurdu.
func runGo(args ...string) (string, error) {
	cmd := exec.Command("go", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// truncateString, uzun yanıtları kısaltır.
func truncateString(s string) string {
	const max = 200
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
