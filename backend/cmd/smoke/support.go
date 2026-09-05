package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// --- kimlik akışı ---

type authResult struct {
	AccessToken    string `json:"accessToken"`
	RefreshToken   string `json:"refreshToken"`
	OrganizationID string `json:"organizationId"`
	User           struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
	Device *struct {
		ID       string `json:"id"`
		Platform string `json:"platform"`
		IsNew    bool   `json:"isNew"`
	} `json:"device"`
}

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

func (r *smokeRunner) requestCode(email string) error {
	_, err := r.mustQuery("", fmt.Sprintf(
		`mutation { requestLoginCode(input:{email:%q}) { sent } }`, email))
	return err
}

func (r *smokeRunner) verifyCode(email, code string, device *deviceInput, signature string) (authResult, error) {
	signatureField := ""
	if signature != "" {
		signatureField = fmt.Sprintf(", deviceSignature:%q", signature)
	}

	resp, err := r.mustQuery("", fmt.Sprintf(
		`mutation { verifyLoginCode(input:{email:%q, code:%q%s%s}) {
			accessToken refreshToken organizationId
			user { id email } device { id platform isNew } } }`,
		email, code, device.render(), signatureField))
	if err != nil {
		return authResult{}, err
	}

	var payload struct {
		VerifyLoginCode authResult `json:"verifyLoginCode"`
	}
	if err := resp.decode(&payload); err != nil {
		return authResult{}, err
	}
	return payload.VerifyLoginCode, nil
}

func (r *smokeRunner) deviceChallenge(email, fingerprint string) (string, error) {
	resp, err := r.mustQuery("", fmt.Sprintf(
		`mutation { requestDeviceChallenge(input:{email:%q, fingerprint:%q}) { challenge } }`,
		email, fingerprint))
	if err != nil {
		return "", err
	}
	var payload struct {
		RequestDeviceChallenge struct {
			Challenge string `json:"challenge"`
		} `json:"requestDeviceChallenge"`
	}
	if err := resp.decode(&payload); err != nil {
		return "", err
	}
	return payload.RequestDeviceChallenge.Challenge, nil
}

func (r *smokeRunner) refreshToken(token string) (authResult, error) {
	resp, err := r.mustQuery("", fmt.Sprintf(
		`mutation { refreshToken(input:{refreshToken:%q}) {
			accessToken refreshToken organizationId user { id email } } }`, token))
	if err != nil {
		return authResult{}, err
	}
	var payload struct {
		RefreshToken authResult `json:"refreshToken"`
	}
	if err := resp.decode(&payload); err != nil {
		return authResult{}, err
	}
	return payload.RefreshToken, nil
}

// loginAdmin, tohum yöneticisiyle giriş yapar.
func (r *smokeRunner) loginAdmin() (string, error) {
	const adminEmail = "yonetici@prova.local"

	if err := r.requestCode(adminEmail); err != nil {
		return "", err
	}
	body, err := r.mailBody(adminEmail)
	if err != nil {
		return "", err
	}
	code, ok := loginCode(body)
	if !ok {
		return "", fmt.Errorf("yönetici iletisinde kod yok")
	}
	auth, err := r.verifyCode(adminEmail, code, nil, "")
	if err != nil {
		return "", err
	}
	return auth.AccessToken, nil
}

// joinDemoOrg, duman testi kullanıcısını tohum organizasyonuna taşır.
//
// Yeni kullanıcı kendi kişisel organizasyonunda açılıyor ve orada oynanabilir
// içerik yok. Kişisel üyelik siliniyor çünkü çözücü en eski etkin üyeliği
// seçiyor ve o, kişisel organizasyon.
func (r *smokeRunner) joinDemoOrg(userID string) error {
	statements := []string{
		fmt.Sprintf(`INSERT INTO organization_users (organization_id, user_id, status) VALUES ('%s','%s','active') ON CONFLICT (organization_id, user_id) DO UPDATE SET status='active'`, demoOrg, userID),
		fmt.Sprintf(`DELETE FROM organization_users WHERE user_id='%s' AND organization_id <> '%s'`, userID, demoOrg),
		fmt.Sprintf(`INSERT INTO user_roles (user_id, role_id, organization_id) SELECT '%s', id, '%s' FROM roles WHERE scope_id='%s' AND name='trainee'`, userID, demoOrg, demoOrg),
	}
	for _, sql := range statements {
		if _, err := psql(sql); err != nil {
			return err
		}
	}
	return nil
}

// --- oturum akışı ---

func (r *smokeRunner) startSession(token, scenario string) (string, error) {
	resp, err := r.mustQuery(token, fmt.Sprintf(
		`mutation { startSession(input:{scenarioLineageId:%q}) { id status } }`, scenario))
	if err != nil {
		return "", err
	}
	var payload struct {
		StartSession struct {
			ID string `json:"id"`
		} `json:"startSession"`
	}
	if err := resp.decode(&payload); err != nil {
		return "", err
	}
	return payload.StartSession.ID, nil
}

// submitTurn, mesajı gönderir ve karakterin yanıtını döndürür.
func (r *smokeRunner) submitTurn(token, sessionID, text string) (string, error) {
	resp, err := r.mustQuery(token, fmt.Sprintf(
		`mutation { submitTurn(input:{sessionId:%q, text:%q}) { index role text } }`,
		sessionID, text))
	if err != nil {
		return "", err
	}
	var payload struct {
		SubmitTurn struct {
			Text string `json:"text"`
		} `json:"submitTurn"`
	}
	if err := resp.decode(&payload); err != nil {
		return "", err
	}
	return payload.SubmitTurn.Text, nil
}

func (r *smokeRunner) endSession(token, sessionID string) error {
	_, err := r.mustQuery(token, fmt.Sprintf(
		`mutation { endSession(sessionId:%q) { id status } }`, sessionID))
	return err
}

type scoreView struct {
	ID       string  `json:"id"`
	Total    float64 `json:"total"`
	MaxTotal float64 `json:"maxTotal"`
	Passed   bool    `json:"passed"`
	Model    string  `json:"model"`
	Criteria []struct {
		CriterionKey  string `json:"criterionKey"`
		Quote         string `json:"quote"`
		QuoteVerified bool   `json:"quoteVerified"`
	} `json:"criteria"`
	Override *struct {
		Reason        string  `json:"reason"`
		PreviousTotal float64 `json:"previousTotal"`
	} `json:"override"`
}

func (r *smokeRunner) sessionScore(token, sessionID string) (scoreView, error) {
	resp, err := r.mustQuery(token, fmt.Sprintf(
		`{ session(id:%q) { score {
			id total maxTotal passed model
			criteria { criterionKey quote quoteVerified }
			override { reason previousTotal }
		} } }`, sessionID))
	if err != nil {
		return scoreView{}, err
	}
	var payload struct {
		Session struct {
			Score *scoreView `json:"score"`
		} `json:"session"`
	}
	if err := resp.decode(&payload); err != nil {
		return scoreView{}, err
	}
	if payload.Session.Score == nil {
		return scoreView{}, fmt.Errorf("oturumun puanı yok")
	}
	return *payload.Session.Score, nil
}

type turnView struct {
	Index int    `json:"index"`
	Role  string `json:"role"`
	Text  string `json:"text"`
}

func (r *smokeRunner) sessionTurns(token, sessionID string) ([]turnView, error) {
	resp, err := r.mustQuery(token, fmt.Sprintf(
		`{ session(id:%q) { turns { index role text } } }`, sessionID))
	if err != nil {
		return nil, err
	}
	var payload struct {
		Session struct {
			Turns []turnView `json:"turns"`
		} `json:"session"`
	}
	if err := resp.decode(&payload); err != nil {
		return nil, err
	}
	return payload.Session.Turns, nil
}

func (r *smokeRunner) overrideScore(token, scoreID string, total float64, passed bool, reason string) (scoreView, error) {
	resp, err := r.mustQuery(token, fmt.Sprintf(
		`mutation { overrideScore(input:{scoreId:%q, total:%v, passed:%t, reason:%q}) {
			id total maxTotal passed model
			criteria { criterionKey quote quoteVerified }
			override { reason previousTotal }
		} }`, scoreID, total, passed, reason))
	if err != nil {
		return scoreView{}, err
	}
	var payload struct {
		OverrideScore scoreView `json:"overrideScore"`
	}
	if err := resp.decode(&payload); err != nil {
		return scoreView{}, err
	}
	return payload.OverrideScore, nil
}

// --- hesap yaşam döngüsü ---

func (r *smokeRunner) exportData(token string) (map[string]any, error) {
	resp, err := r.mustQuery(token, `mutation { exportMyData { document } }`)
	if err != nil {
		return nil, err
	}
	var payload struct {
		ExportMyData struct {
			Document map[string]any `json:"document"`
		} `json:"exportMyData"`
	}
	if err := resp.decode(&payload); err != nil {
		return nil, err
	}
	return payload.ExportMyData.Document, nil
}

func (r *smokeRunner) requestDeletion(token string) error {
	_, err := r.mustQuery(token, `mutation { requestAccountDeletion { requestedAt scheduledAt } }`)
	return err
}

// waitForPurge, zamanlanmış silme işinin hesabı almasını bekler.
func (r *smokeRunner) waitForPurge(userID string) error {
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		status, err := psql(fmt.Sprintf(`SELECT status FROM users WHERE id='%s'`, userID))
		if err != nil {
			return err
		}
		if status == "deleted" {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("kalıcı silme işi 90 saniyede tetiklenmedi; LIFECYCLE_PURGE_INTERVAL_SECONDS kısa tutulmalı")
}

// --- yardımcılar ---

func psql(sql string) (string, error) {
	cmd := exec.Command("docker", "exec", "prova-postgres",
		"psql", "-U", "masterfabric", "-d", "masterfabric", "-tAc", sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("psql: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func mongo(js string) (string, error) {
	cmd := exec.Command("docker", "exec", "prova-mongo", "mongosh", "--quiet", "prova", "--eval", js)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("mongosh: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

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
