package main

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

// Tohum verisindeki senaryo. Duman testi tohum içeriğini oynuyor: kendi
// içeriğini üretseydi, tohumun oynanabilir olduğunu hiç sınamazdı.
const smokeScenario = "77777777-7777-4777-8777-777777777777"

// demoOrg, tohum organizasyonu.
const demoOrg = "11111111-1111-4111-8111-111111111111"

// run, zincirin tamamını yürütür.
//
// Her adım bir öncekinin çıktısına bağlı ve ilk hata zinciri kesiyor:
// "skoru al" adımı, "oturumu bitir" adımı gerçekten çalışmadıysa anlamsız.
func (r *smokeRunner) run() error {
	started := time.Now()
	defer func() { r.elapsed = time.Since(started) }()

	email := fmt.Sprintf("duman-%s@prova.local", r.runID)
	fingerprint := fmt.Sprintf("smoke-fp-%s-0123456789", r.runID)

	keys, err := newKeyPair()
	if err != nil {
		return fmt.Errorf("anahtar çifti üretilemedi: %w", err)
	}

	// --- 1. Kayıt ve ayrı e-posta doğrulaması ---
	if err := r.registerAndVerify(email); err != nil {
		return fmt.Errorf("kayıt/e-posta doğrulama başarısız: %w", err)
	}
	r.record("kayıt ve e-posta doğrulama", email)

	// --- 2. Giriş kodu iste ---
	if err := r.requestCode(email); err != nil {
		return fmt.Errorf("kod istenemedi: %w", err)
	}
	body, err := r.mailBody(email)
	if err != nil {
		return fmt.Errorf("e-posta okunamadı: %w", err)
	}
	code, ok := loginCode(body)
	if !ok {
		return fmt.Errorf("iletide kod yok")
	}
	link, hasLink := magicLink(body)
	linkNote := "yalnızca kod"
	if hasLink {
		linkNote = "kod + magic link"
	}
	r.record("kod istendi", fmt.Sprintf("%s (%s)", email, linkNote))
	_ = link

	// --- 3. Giriş yap ve cihaz kaydet ---
	auth, err := r.verifyCode(email, code, &deviceInput{
		Fingerprint: fingerprint,
		Name:        "Duman testi cihazı",
		Platform:    "darwin",
		PublicKey:   keys.publicB64,
	}, "")
	if err != nil {
		return fmt.Errorf("kod doğrulanamadı: %w", err)
	}
	if auth.Device == nil {
		return fmt.Errorf("cihaz eşleşmedi")
	}
	r.record("kod doğrulandı, cihaz kaydedildi",
		fmt.Sprintf("platform %s, yeni=%t", auth.Device.Platform, auth.Device.IsNew))

	// --- 4. Cihaz challenge'ı ile ikinci giriş ---
	challenge, err := r.deviceChallenge(email, fingerprint)
	if err != nil {
		return fmt.Errorf("challenge alınamadı: %w", err)
	}
	if err := r.joinDemoOrg(auth.User.ID); err != nil {
		return fmt.Errorf("demo organizasyonuna alınamadı: %w", err)
	}
	if err := r.requestCode(email); err != nil {
		return fmt.Errorf("ikinci kod istenemedi: %w", err)
	}
	body, err = r.mailBody(email)
	if err != nil {
		return err
	}
	code, ok = loginCode(body)
	if !ok {
		return fmt.Errorf("ikinci iletide kod yok")
	}
	auth, err = r.verifyCode(email, code, &deviceInput{
		Fingerprint: fingerprint, Platform: "darwin",
	}, keys.sign(challenge))
	if err != nil {
		return fmt.Errorf("challenge imzasıyla giriş başarısız: %w", err)
	}
	if auth.OrganizationID != demoOrg {
		return fmt.Errorf("demo organizasyonu beklenirdi, %s geldi", auth.OrganizationID)
	}
	token := auth.AccessToken
	refresh := auth.RefreshToken
	r.record("cihaz challenge'ı imzalanarak girildi",
		fmt.Sprintf("org %s…, refresh token alındı", auth.OrganizationID[:8]))

	// --- 4. Oturum başlat ---
	session, err := r.startSession(token, smokeScenario)
	if err != nil {
		return fmt.Errorf("oturum başlatılamadı: %w", err)
	}
	r.record("oturum başlatıldı", session[:8]+"…")

	// --- 5. Birkaç konuşma sırası ---
	messages := []string{
		"Merhaba, ben müşteri hizmetlerinden Ayşe. Size nasıl yardımcı olabilirim?",
		"Anlıyorum. İade koşullarımız için sipariş numaranızı alabilir miyim?",
		"Teşekkürler. Ürün kullanılmamışsa 30 gün içinde ücretsiz iade alabiliyoruz.",
	}
	for i, message := range messages {
		reply, err := r.submitTurn(token, session, message)
		if err != nil {
			return fmt.Errorf("%d. konuşma sırası gönderilemedi: %w", i+1, err)
		}
		if strings.TrimSpace(reply) == "" {
			return fmt.Errorf("%d. sırada karakter yanıt vermedi", i+1)
		}
	}
	turns, err := r.sessionTurns(token, session)
	if err != nil {
		return err
	}
	r.record("konuşma sıraları gönderildi",
		fmt.Sprintf("%d mesaj, transkript %d sıra", len(messages), len(turns)))

	// --- 6. Oturumu bitir ---
	if err := r.endSession(token, session); err != nil {
		return fmt.Errorf("oturum bitirilemedi: %w", err)
	}
	r.record("oturum bitirildi ve puanlandı", "")

	// --- 7. Skoru al ---
	score, err := r.sessionScore(token, session)
	if err != nil {
		return fmt.Errorf("skor alınamadı: %w", err)
	}
	outcome := "GEÇTİ"
	if !score.Passed {
		outcome = "KALDI"
	}
	r.record("skor alındı",
		fmt.Sprintf("%.1f/%.1f → %s (model %s)", score.Total, score.MaxTotal, outcome, score.Model))

	// --- 8. Alıntıları doğrula ---
	verified, fabricated := 0, 0
	for _, criterion := range score.Criteria {
		if criterion.QuoteVerified {
			verified++
		} else if strings.TrimSpace(criterion.Quote) != "" {
			fabricated++
		}
	}
	if verified == 0 {
		return fmt.Errorf("hiçbir alıntı doğrulanmadı; alıntı doğrulaması çalışmıyor")
	}
	// Alıntının transkriptte GERÇEKTEN geçtiğini bağımsız olarak sınıyoruz:
	// quoteVerified alanına güvenmek, doğrulamayı doğrulanan şeyin kendisine
	// sormak olurdu.
	if err := r.confirmQuotesAppearInTranscript(score, turns); err != nil {
		return err
	}
	r.record("alıntılar doğrulandı",
		fmt.Sprintf("%d/%d transkriptte bulundu, %d uydurma işaretlendi",
			verified, len(score.Criteria), fabricated))

	// --- 9. Skoru ez ---
	admin, err := r.loginAdmin()
	if err != nil {
		return fmt.Errorf("yönetici girişi başarısız: %w", err)
	}
	overridden, err := r.overrideScore(admin, score.ID, 30, true, "Duman testi: itiraz sonrası düzeltme")
	if err != nil {
		return fmt.Errorf("skor ezilemedi: %w", err)
	}
	if overridden.Override == nil {
		return fmt.Errorf("ezme kaydı yazılmadı")
	}
	r.record("skor ezildi",
		fmt.Sprintf("%.1f → %.1f, eski sonuç saklandı", overridden.Override.PreviousTotal, overridden.Total))

	// Çalışan ezme kaydını görmemeli.
	traineeScore, err := r.sessionScore(token, session)
	if err != nil {
		return err
	}
	if traineeScore.Override != nil {
		return fmt.Errorf("çalışan ezme kaydını görüyor; izin directive'i çalışmıyor")
	}
	r.record("ezme kaydı çalışandan gizli", "override alanı null")

	// --- 10. Token yenileme (rotasyon) ---
	rotated, err := r.refreshToken(refresh)
	if err != nil {
		return fmt.Errorf("token yenilenemedi: %w", err)
	}
	if rotated.RefreshToken == refresh {
		return fmt.Errorf("rotasyon yeni token üretmedi")
	}
	token = rotated.AccessToken
	r.record("token yenilendi (rotasyon)", "yeni access + refresh çifti")

	// --- 11. Veriyi dışa aktar ---
	export, err := r.exportData(token)
	if err != nil {
		return fmt.Errorf("veri dışa aktarılamadı: %w", err)
	}
	sessions, _ := export["sessions"].([]any)
	if len(sessions) == 0 {
		return fmt.Errorf("dışa aktarmada oturum yok")
	}
	first, _ := sessions[0].(map[string]any)
	if _, hasTranscript := first["transcript"]; !hasTranscript {
		return fmt.Errorf("dışa aktarmada transkript yok")
	}
	if _, hasScore := first["score"]; !hasScore {
		return fmt.Errorf("dışa aktarmada puan yok")
	}
	r.record("veri dışa aktarıldı",
		fmt.Sprintf("%d oturum, transkript ve puan dâhil", len(sessions)))

	// --- 12. Hesabı sil ---
	if err := r.requestDeletion(token); err != nil {
		return fmt.Errorf("silme talebi verilemedi: %w", err)
	}
	status, err := psql(fmt.Sprintf(`SELECT status FROM users WHERE id='%s'`, auth.User.ID))
	if err != nil {
		return err
	}
	if status != "pending_deletion" {
		return fmt.Errorf("pending_deletion bekleniyordu, %q geldi", status)
	}
	r.record("hesap silme talebi verildi", "durum pending_deletion")

	// --- 13. Kalıcı silmeyi tetikle ---
	//
	// Bekleme süresini beklemek yerine tarih geçmişe çekiliyor. İşin kendisi
	// zaten "tarihi geçmiş olanları sil" mantığıyla çalışıyor, ve duman
	// testinin otuz gün beklemesi kabul edilemez.
	if _, err := psql(fmt.Sprintf(
		`UPDATE users SET deletion_scheduled_at = NOW() - INTERVAL '1 hour' WHERE id='%s'`,
		auth.User.ID)); err != nil {
		return err
	}
	if err := r.waitForPurge(auth.User.ID); err != nil {
		return err
	}

	row, err := psql(fmt.Sprintf(
		`SELECT COALESCE(email,'<null>'), status FROM users WHERE id='%s'`, auth.User.ID))
	if err != nil {
		return err
	}
	if !strings.HasPrefix(row, "<null>|deleted") {
		return fmt.Errorf("silme sonrası beklenen '<null>|deleted', gelen %q", row)
	}
	r.record("kalıcı silme tamamlandı", "e-posta null, durum deleted")

	// --- 14. Oturum kimliksizleştirildi ama duruyor ---
	anonymized, err := mongo(fmt.Sprintf(
		`JSON.stringify(db.sessions.findOne({_id: UUID("%s")}, {anonymized:1, _id:0}))`, session))
	if err != nil {
		return err
	}
	if !strings.Contains(strings.ReplaceAll(anonymized, " ", ""), `"anonymized":true`) {
		return fmt.Errorf("oturum kimliksizleştirilmedi: %s", anonymized)
	}
	scores, err := mongo(fmt.Sprintf(`db.scores.countDocuments({session_id: UUID("%s")})`, session))
	if err != nil {
		return err
	}
	if strings.TrimSpace(scores) == "0" {
		return fmt.Errorf("puan kaydı silinmiş; kurum istatistiği için korunmalıydı")
	}
	auditCount, err := psql(fmt.Sprintf(
		`SELECT count(*) FROM audit_logs WHERE user_id='%s'`, auth.User.ID))
	if err != nil {
		return err
	}
	r.record("oturum kimliksiz, puan ve denetim korundu",
		fmt.Sprintf("%s puan, %s denetim kaydı", strings.TrimSpace(scores), auditCount))

	// --- 15. Silinen hesabın token'ı artık çalışmamalı ---
	resp, err := r.query(token, `{ me { email } }`)
	if err != nil {
		return err
	}
	if len(resp.Errors) == 0 && resp.status < 400 {
		var payload struct {
			Me *struct {
				Email string `json:"email"`
			} `json:"me"`
		}
		_ = resp.decode(&payload)
		if payload.Me != nil && payload.Me.Email != "" {
			return fmt.Errorf("silinen hesap hâlâ e-posta döndürüyor: %s", payload.Me.Email)
		}
	}
	r.record("silinen hesabın kimliği boş", "e-posta artık dönmüyor")

	_ = base64.StdEncoding
	return nil
}

// confirmQuotesAppearInTranscript, doğrulanmış alıntıların transkriptte
// gerçekten geçtiğini bağımsız olarak sınar.
//
// quoteVerified alanına güvenmek, doğrulamayı doğrulanan şeyin kendisine
// sormak olurdu: alan her zaman true dönse de test geçerdi.
func (r *smokeRunner) confirmQuotesAppearInTranscript(score scoreView, turns []turnView) error {
	normalize := func(s string) string {
		return strings.Join(strings.Fields(strings.ToLower(s)), " ")
	}

	transcript := make([]string, 0, len(turns))
	for _, t := range turns {
		transcript = append(transcript, normalize(t.Text))
	}

	for _, criterion := range score.Criteria {
		if !criterion.QuoteVerified {
			continue
		}
		needle := normalize(criterion.Quote)
		if needle == "" {
			return fmt.Errorf("%s kriteri boş alıntıyla doğrulanmış işaretlenmiş", criterion.CriterionKey)
		}
		found := false
		for _, text := range transcript {
			if strings.Contains(text, needle) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%s kriterinin alıntısı doğrulanmış işaretlenmiş ama transkriptte yok: %q",
				criterion.CriterionKey, criterion.Quote)
		}
	}
	return nil
}
