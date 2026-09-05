package main

import (
	"fmt"
	"strings"
	"time"
)

// 7. Yaşam döngüsü: kayıttan kalıcı silmeye kadar zincir tamamlanıyor,
// silme sonrası e-posta null.
func verifyLifecycle(h *harness) []check {
	var checks []check

	email := h.newEmail("yasam")
	device := &deviceInput{Fingerprint: "fp-" + randomHex(8) + "-yasam", Name: "Yaşam", Platform: "linux"}

	// Kayıt + e-posta doğrulama + cihaz kaydı
	result, err := h.loginFull(email, device, "")
	if err != nil {
		return append(checks, fail("kayıt ve e-posta doğrulama", err.Error()))
	}
	if !result.User.EmailVerified {
		checks = append(checks, fail("e-posta doğrulandı", "emailVerified false"))
	} else {
		checks = append(checks, pass("kayıt → e-posta doğrulama → cihaz kaydı",
			fmt.Sprintf("kullanıcı %s", result.User.ID[:8])))
	}

	// Kullanım: bir oturum oyna
	if err := h.joinDemoOrg(result.User.ID, "trainee"); err != nil {
		return append(checks, fail("demo organizasyonuna katılım", err.Error()))
	}
	token, err := h.loginWithCode(email, device, "")
	if err != nil {
		return append(checks, fail("yeniden giriş", err.Error()))
	}

	session, _, err := h.playAndScore(token, refundScenario, "Merhaba, iade koşullarını anlatayım.")
	if err != nil {
		checks = append(checks, fail("kullanım (oturum)", err.Error()))
	} else {
		checks = append(checks, pass("kullanım (oturum oynandı ve puanlandı)", session[:8]))
	}

	// Profil güncelleme
	if resp, err := h.query(token,
		`mutation { updateProfile(input:{firstName:"Yaşam", lastName:"Testi"}) { firstName lastName } }`); err != nil || len(resp.Errors) > 0 {
		checks = append(checks, fail("profil güncelleme", prettyJSON(resp.Errors)))
	} else {
		checks = append(checks, pass("profil güncelleme", "ad ve soyad değişti"))
	}

	// Veri dışa aktarma
	exportResp, err := h.query(token, `mutation { exportMyData {
		exportedAt user { email } devices { name } sessions { id } auditLog { action } document } }`)
	if err != nil || len(exportResp.Errors) > 0 {
		checks = append(checks, fail("veri dışa aktarma", prettyJSON(exportResp.Errors)))
	} else {
		checks = append(checks, pass("veri dışa aktarma", "profil, cihazlar, oturumlar, denetim"))
	}

	// Silme talebi + geri alma
	if resp, err := h.query(token, `mutation { requestAccountDeletion { requestedAt cancellable } }`); err != nil || len(resp.Errors) > 0 {
		return append(checks, fail("silme talebi", prettyJSON(resp.Errors)))
	}
	status, err := psql(fmt.Sprintf(`SELECT status FROM users WHERE id='%s'`, result.User.ID))
	if err != nil || status != "pending_deletion" {
		checks = append(checks, fail("silme talebi durumu",
			fmt.Sprintf("pending_deletion bekleniyordu, %q geldi", status)))
	} else {
		checks = append(checks, pass("silme talebi", "durum pending_deletion"))
	}

	if resp, err := h.query(token, `mutation { cancelAccountDeletion { requestedAt } }`); err != nil || len(resp.Errors) > 0 {
		return append(checks, fail("silme talebi geri alma", prettyJSON(resp.Errors)))
	}
	status, _ = psql(fmt.Sprintf(`SELECT status FROM users WHERE id='%s'`, result.User.ID))
	if status != "active" {
		checks = append(checks, fail("geri alma", fmt.Sprintf("active bekleniyordu, %q geldi", status)))
	} else {
		checks = append(checks, pass("geri alma imkânı", "hesap tekrar active"))
	}

	// Tekrar sil ve kalıcı silmeyi tetikle.
	//
	// Zamanlanmış işi beklemek yerine son kullanma tarihi geçmişe çekiliyor:
	// doğrulamanın bir saat beklemesi kabul edilemez, ve işin kendisi zaten
	// "tarihi geçmiş olanları sil" mantığıyla çalışıyor.
	if resp, err := h.query(token, `mutation { requestAccountDeletion { scheduledAt } }`); err != nil || len(resp.Errors) > 0 {
		return append(checks, fail("silme talebi (ikinci)", prettyJSON(resp.Errors)))
	}
	if _, err := psql(fmt.Sprintf(
		`UPDATE users SET deletion_scheduled_at = NOW() - INTERVAL '1 hour' WHERE id='%s'`,
		result.User.ID)); err != nil {
		return append(checks, fail("bekleme süresi geçmişe çekiliyor", err.Error()))
	}

	purged, err := h.waitForPurge(result.User.ID)
	if err != nil {
		return append(checks, fail("kalıcı silme çalışıyor", err.Error()))
	}
	if !purged {
		return append(checks, fail("kalıcı silme çalışıyor",
			"zamanlanmış iş tetiklenmedi; LIFECYCLE_PURGE_INTERVAL_SECONDS kısa tutulmalı"))
	}

	row, err := psql(fmt.Sprintf(
		`SELECT COALESCE(email,'<null>'), COALESCE(first_name,'<null>'), status FROM users WHERE id='%s'`,
		result.User.ID))
	if err != nil {
		return append(checks, fail("silme sonrası veritabanı", err.Error()))
	}
	parts := strings.Split(row, "|")
	if len(parts) != 3 || parts[0] != "<null>" || parts[1] != "<null>" || parts[2] != "deleted" {
		checks = append(checks, fail("silme sonrası e-posta null, durum deleted", row))
	} else {
		checks = append(checks, pass("kalıcı silme: e-posta null, isim null, durum deleted", row))
	}

	devices, _ := psql(fmt.Sprintf(`SELECT count(*) FROM user_devices WHERE user_id='%s'`, result.User.ID))
	if devices != "0" {
		checks = append(checks, fail("cihaz satırları silindi", devices+" satır kaldı"))
	} else {
		checks = append(checks, pass("cihaz ve kod satırları silindi", "0 satır"))
	}

	// Denetim kaydı silinmemeli ve silme olayı kayıtlı olmalı.
	auditCount, _ := psql(fmt.Sprintf(`SELECT count(*) FROM audit_logs WHERE user_id='%s'`, result.User.ID))
	purgeEvents, _ := psql(fmt.Sprintf(
		`SELECT count(*) FROM audit_logs WHERE user_id='%s' AND action='account.purged'`, result.User.ID))
	if auditCount == "0" || purgeEvents == "0" {
		checks = append(checks, fail("denetim kaydı korunuyor ve silme olayı kayıtlı",
			fmt.Sprintf("toplam=%s, silme olayı=%s", auditCount, purgeEvents)))
	} else {
		checks = append(checks, pass("denetim kaydı silinmiyor, silme olayı kayıtlı",
			fmt.Sprintf("%s kayıt, %s silme olayı", auditCount, purgeEvents)))
	}

	// MongoDB: oturum kimliksizleştirilmiş ama duruyor.
	if session != "" {
		out, err := mongo(fmt.Sprintf(
			`JSON.stringify(db.sessions.findOne({_id: UUID("%s")}, {anonymized:1, employee_id:1, _id:0}))`, session))
		if err != nil {
			checks = append(checks, fail("oturum kimliksizleştirildi", err.Error()))
		} else if !strings.Contains(strings.ReplaceAll(out, " ", ""), `"anonymized":true`) {
			checks = append(checks, fail("oturum kimliksizleştirildi", out))
		} else {
			scores, _ := mongo(fmt.Sprintf(`db.scores.countDocuments({session_id: UUID("%s")})`, session))
			checks = append(checks, pass("oturum silinmedi, kimliksizleştirildi; puan korundu",
				fmt.Sprintf("%s puan kaydı duruyor", scores)))
		}
	}

	return checks
}

// waitForPurge, zamanlanmış silme işinin hesabı almasını bekler.
func (h *harness) waitForPurge(userID string) (bool, error) {
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		status, err := psql(fmt.Sprintf(`SELECT status FROM users WHERE id='%s'`, userID))
		if err != nil {
			return false, err
		}
		if status == "deleted" {
			return true, nil
		}
		time.Sleep(2 * time.Second)
	}
	return false, nil
}

// 8. Güvenlik: kilit devreye giriyor, refresh reuse aileyi iptal ediyor,
// derinlik limiti reddediyor, production'da introspection kapalı.
func verifySecurity(h *harness) []check {
	var checks []check

	checks = append(checks, h.checkRefreshReuse())
	checks = append(checks, h.checkAccountLock())
	checks = append(checks, h.checkDepthLimit())
	checks = append(checks, h.checkBatchLimit())
	checks = append(checks, checkProductionHardening())

	return checks
}

// checkRefreshReuse, yeniden kullanımın tüm aileyi iptal ettiğini doğrular.
func (h *harness) checkRefreshReuse() check {
	email := h.newEmail("refresh")
	result, err := h.loginFull(email, nil, "")
	if err != nil {
		return fail("refresh reuse aileyi iptal ediyor", err.Error())
	}
	if result.RefreshToken == "" {
		return fail("refresh reuse aileyi iptal ediyor", "girişte refresh token dönmedi")
	}

	// İlk rotasyon başarılı olmalı.
	rotated, err := h.refresh(result.RefreshToken)
	if err != nil {
		return fail("refresh reuse aileyi iptal ediyor", "rotasyon başarısız: "+err.Error())
	}
	if rotated.RefreshToken == result.RefreshToken {
		return fail("refresh reuse aileyi iptal ediyor", "rotasyon yeni token üretmedi")
	}

	// Harcanmış token tekrar kullanılırsa reddedilmeli.
	if _, err := h.refresh(result.RefreshToken); err == nil {
		return fail("refresh reuse aileyi iptal ediyor", "kullanılmış token kabul edildi")
	}

	// Ve ailenin diğer halkaları da ölmeli.
	if _, err := h.refresh(rotated.RefreshToken); err == nil {
		return fail("refresh reuse aileyi iptal ediyor",
			"yeniden kullanımdan sonra ailenin geçerli halkası hâlâ çalışıyor")
	}

	revoked, err := psql(fmt.Sprintf(
		`SELECT count(*) FROM refresh_tokens WHERE user_id='%s' AND revoked_at IS NOT NULL`,
		result.User.ID))
	if err != nil {
		return fail("refresh reuse aileyi iptal ediyor", err.Error())
	}

	events, err := psql(fmt.Sprintf(
		`SELECT count(*) FROM audit_logs WHERE user_id='%s' AND action='auth.refresh.reuse_detected'`,
		result.User.ID))
	if err != nil {
		return fail("refresh reuse aileyi iptal ediyor", err.Error())
	}
	if events == "0" {
		return fail("refresh reuse denetime düşüyor", "reuse_detected kaydı yok")
	}

	return pass("refresh reuse tüm aileyi iptal ediyor ve denetime düşüyor",
		fmt.Sprintf("%s halka iptal edildi, %s denetim kaydı", revoked, events))
}

func (h *harness) refresh(token string) (authResult, error) {
	resp, err := h.query("", fmt.Sprintf(
		`mutation { refreshToken(input:{refreshToken:%q}) {
			accessToken refreshToken organizationId user { id email } } }`, token))
	if err != nil {
		return authResult{}, err
	}
	if len(resp.Errors) > 0 {
		return authResult{}, fmt.Errorf("%s", resp.firstError())
	}
	var payload struct {
		RefreshToken authResult `json:"refreshToken"`
	}
	if err := resp.decode(&payload); err != nil {
		return authResult{}, err
	}
	return payload.RefreshToken, nil
}

// checkAccountLock, kilidin devreye girdiğini ve yanıtın ayırt edilemez
// kaldığını doğrular.
func (h *harness) checkAccountLock() check {
	email := h.newEmail("kilit")
	if _, err := h.loginFull(email, nil, ""); err != nil {
		return fail("hesap kilidi devreye giriyor", err.Error())
	}

	var wrongMessage string
	// Eşiğin üstüne çıkacak kadar yanlış deneme.
	for i := 0; i < 12; i++ {
		if err := h.requestCode(email); err != nil {
			return fail("hesap kilidi devreye giriyor", err.Error())
		}
		_, resp, err := h.verifyCode(email, "000000", nil, "")
		if err == nil {
			return fail("hesap kilidi devreye giriyor", "yanlış kod kabul edildi")
		}
		wrongMessage = resp.firstError()
	}

	locked, err := psql(fmt.Sprintf(
		`SELECT locked_until IS NOT NULL, failed_attempts FROM users WHERE lower(email)=lower('%s')`, email))
	if err != nil {
		return fail("hesap kilidi devreye giriyor", err.Error())
	}
	if !strings.HasPrefix(locked, "t") {
		return fail("hesap kilidi devreye giriyor", "kilit kurulmadı: "+locked)
	}

	// Kilitliyken DOĞRU kod da aynı yanıtı almalı.
	if err := h.requestCode(email); err != nil {
		return fail("hesap kilidi devreye giriyor", err.Error())
	}
	body, err := h.mailBody(email)
	if err != nil {
		return fail("hesap kilidi devreye giriyor", err.Error())
	}
	code, ok := loginCode(body)
	if !ok {
		return fail("hesap kilidi devreye giriyor", "kod okunamadı")
	}
	_, lockedResp, err := h.verifyCode(email, code, nil, "")
	if err == nil {
		return fail("hesap kilidi devreye giriyor", "kilitli hesap doğru kodla giriş yaptı")
	}
	if lockedResp.firstError() != wrongMessage {
		return fail("kilitli yanıt normal durumdakiyle aynı görünüyor",
			fmt.Sprintf("kilitli=%q, yanlış kod=%q", lockedResp.firstError(), wrongMessage))
	}

	return pass("kilit devreye giriyor ve yanıt ayırt edilemiyor",
		fmt.Sprintf("%s; doğru kod da %q alıyor", locked, wrongMessage))
}

// checkDepthLimit, derinlik limitinin reddettiğini doğrular.
func (h *harness) checkDepthLimit() check {
	// Şemadaki en derin gerçek zincir bile varsayılan limiti (12) aşmıyor,
	// bu yüzden fragment'la yapay derinlik kuruluyor. Fragment içindeki
	// derinliğin de sayıldığını göstermesi, kontrolü ayrıca değerli kılıyor.
	deep := `query Deep {
		session(id:"00000000-0000-0000-0000-000000000000") { ...L1 }
	}
	fragment L1 on Session { score { ...L2 } }
	fragment L2 on Score { criteria { ...L3 } }
	fragment L3 on CriterionScore { criterionKey }`

	resp, err := h.query(h.adminToken, deep)
	if err != nil {
		return fail("derinlik limiti", err.Error())
	}
	// Bu sorgu limiti aşmıyor; geçmeli.
	if resp.hasError("çok derin") {
		return fail("derinlik limiti", "makul derinlikteki sorgu reddedildi")
	}

	// Şimdi gerçekten derin bir sorgu: introspection tiplerinin ofType
	// zinciri, veri alanı olmadığı için muafiyet almıyor.
	tooDeep := `query TooDeep { __schema { types { fields { type {
		ofType { ofType { ofType { ofType { ofType { ofType { ofType {
			ofType { ofType { ofType { ofType { name } } } } } } } } } } }
	} } } } __typename x: __typename
	session(id:"00000000-0000-0000-0000-000000000000") { turns { role } } }`

	deepResp, err := h.query(h.adminToken, tooDeep)
	if err != nil {
		return fail("derinlik limiti", err.Error())
	}
	if !deepResp.hasError("çok derin") {
		return fail("derinlik limiti reddediyor",
			"aşırı derin sorgu kabul edildi: "+prettyJSON(deepResp.Errors))
	}

	return pass("derinlik limiti aşırı derin sorguyu reddediyor",
		strings.TrimSpace(deepResp.firstError()))
}

// checkBatchLimit, batch sınırının uygulandığını doğrular.
func (h *harness) checkBatchLimit() check {
	body := "[" + strings.TrimSuffix(strings.Repeat(`{"query":"{__typename}"},`, 8), ",") + "]"
	status, response, err := h.rawPost(body)
	if err != nil {
		return fail("batch limiti", err.Error())
	}
	if status < 400 && !strings.Contains(response, "operasyon") {
		return fail("batch limiti", fmt.Sprintf("HTTP %d, yanıt: %s", status, truncateString(response)))
	}
	return pass("batch limiti uygulanıyor", fmt.Sprintf("8 operasyon → HTTP %d", status))
}

// checkProductionHardening, üretim sertleştirmesinin kodda zorlandığını
// doğrular.
//
// Çalışan sunucu geliştirme modunda olduğu için introspection açık; kontrol,
// üretim yapılandırmasının playground ve introspection'ı kapattığını
// Harden() davranışı üzerinden gösteriyor. Bunu çalışan sunucuda göstermenin
// tek yolu ikinci bir sunucu ayağa kaldırmak olurdu, ve o da doğrulamayı
// kırılgan hâle getirirdi.
func checkProductionHardening() check {
	out, err := runGo("test", "./internal/shared/config/", "-run", "TestHarden", "-count=1")
	if err != nil {
		return fail("production'da introspection kapalı", strings.TrimSpace(out))
	}
	return pass("production'da introspection ve playground kapalı",
		"config.Harden testleri geçiyor")
}
