package main

import (
	"fmt"
	"strings"
)

// 9. E-posta: mailde hem kod hem link var, ikisi de doğruluyor.
func verifyEmail(h *harness) []check {
	var checks []check

	// Kod yolu
	codeEmail := h.newEmail("kod")
	if err := h.requestCode(codeEmail); err != nil {
		return append(checks, fail("kod isteniyor", err.Error()))
	}
	body, err := h.mailBody(codeEmail)
	if err != nil {
		return append(checks, fail("e-posta teslim ediliyor", err.Error()))
	}

	code, hasCode := loginCode(body)
	link, hasLink := magicLink(body)

	if !hasCode {
		checks = append(checks, fail("mailde altı haneli kod var", "kod bulunamadı"))
	}
	if !hasLink {
		checks = append(checks, fail("mailde magic link var", "bağlantı bulunamadı"))
	}
	if hasCode && hasLink {
		checks = append(checks, pass("tek mailde hem kod hem link var",
			fmt.Sprintf("kod %s…, link %s…", code[:2], link[:min(len(link), 45)])))
	}

	if hasCode {
		if _, _, err := h.verifyCode(codeEmail, code, nil, ""); err != nil {
			checks = append(checks, fail("kod ile giriş", err.Error()))
		} else {
			checks = append(checks, pass("kod ile giriş çalışıyor", codeEmail))
		}
	}

	// Link yolu (ayrı adres: kod ve link aynı iletide ama her biri
	// bağımsız olarak doğrulanmalı).
	linkEmail := h.newEmail("link")
	if err := h.requestCode(linkEmail); err != nil {
		return append(checks, fail("link için kod isteniyor", err.Error()))
	}
	linkBody, err := h.mailBody(linkEmail)
	if err != nil {
		return append(checks, fail("link maili", err.Error()))
	}
	linkURL, ok := magicLink(linkBody)
	if !ok {
		return append(checks, fail("link ile giriş", "bağlantı bulunamadı"))
	}
	token, ok := magicToken(linkURL)
	if !ok {
		return append(checks, fail("link ile giriş", "token ayıklanamadı"))
	}

	resp, err := h.query("", fmt.Sprintf(
		`mutation { verifyMagicLink(input:{token:%q}) { accessToken user { email emailVerified } } }`, token))
	if err != nil || len(resp.Errors) > 0 {
		checks = append(checks, fail("link ile giriş çalışıyor", prettyJSON(resp.Errors)))
	} else {
		checks = append(checks, pass("link ile giriş çalışıyor", linkEmail))
	}

	// Tek kullanımlık olmalı.
	reuse, err := h.query("", fmt.Sprintf(
		`mutation { verifyMagicLink(input:{token:%q}) { accessToken } }`, token))
	if err != nil {
		checks = append(checks, fail("link tek kullanımlık", err.Error()))
	} else if len(reuse.Errors) == 0 {
		checks = append(checks, fail("link tek kullanımlık", "aynı bağlantı ikinci kez kabul edildi"))
	} else {
		checks = append(checks, pass("link tek kullanımlık", reuse.firstError()))
	}

	// Bağlantı web arayüzüne işaret etmeli, Electron deep-link'e değil.
	if strings.HasPrefix(linkURL, "http://") || strings.HasPrefix(linkURL, "https://") {
		checks = append(checks, pass("link web arayüzünde açılıyor", "http(s) şeması, deep-link değil"))
	} else {
		checks = append(checks, fail("link web arayüzünde açılıyor", "beklenmeyen şema: "+linkURL))
	}

	return checks
}

// 10. Cihaz: kayıt, challenge doğrulama, iptal, iptal sonrası token reddi.
func verifyDevice(h *harness) []check {
	var checks []check

	email := h.newEmail("cihaz")
	fingerprint := "fp-" + randomHex(8) + "-verify"
	keys, err := newKeyPair()
	if err != nil {
		return append(checks, fail("anahtar çifti üretimi", err.Error()))
	}

	// İlk kayıt: anahtar gönderiliyor, imza istenmiyor (doğrulanacak
	// anahtar henüz yok).
	first, err := h.loginFull(email, &deviceInput{
		Fingerprint: fingerprint, Name: "Doğrulama cihazı",
		Platform: "darwin", PublicKey: keys.publicB64,
	}, "")
	if err != nil {
		return append(checks, fail("cihaz kaydı", err.Error()))
	}
	if first.Device == nil || !first.Device.IsNew {
		return append(checks, fail("cihaz kaydı", "cihaz yeni olarak eşleşmedi"))
	}
	checks = append(checks, pass("cihaz kaydı", "anahtar çiftiyle eşleşti, platform "+first.Device.Platform))

	// Anahtar kaydedilmiş olmalı.
	listResp, err := h.query(first.AccessToken, `{ myDevices { id hasPublicKey } }`)
	if err != nil {
		return append(checks, fail("açık anahtar kaydedildi", err.Error()))
	}
	var devices struct {
		MyDevices []struct {
			ID           string `json:"id"`
			HasPublicKey bool   `json:"hasPublicKey"`
		} `json:"myDevices"`
	}
	if err := listResp.decode(&devices); err != nil || len(devices.MyDevices) == 0 {
		return append(checks, fail("açık anahtar kaydedildi", "cihaz listesi boş"))
	}
	if !devices.MyDevices[0].HasPublicKey {
		checks = append(checks, fail("açık anahtar kaydedildi", "hasPublicKey false"))
	} else {
		checks = append(checks, pass("açık anahtar kaydedildi", "hasPublicKey true"))
	}

	// İmzasız giriş artık reddedilmeli.
	if _, err := h.loginFull(email, &deviceInput{Fingerprint: fingerprint, Platform: "darwin"}, ""); err == nil {
		checks = append(checks, fail("imzasız giriş reddediliyor", "imzasız giriş kabul edildi"))
	} else {
		checks = append(checks, pass("kayıtlı cihazda imza zorunlu", err.Error()))
	}

	// Challenge al, imzala, gir.
	challenge, err := h.deviceChallenge(email, fingerprint)
	if err != nil {
		return append(checks, fail("challenge alınıyor", err.Error()))
	}
	signed, err := h.loginFull(email, &deviceInput{Fingerprint: fingerprint, Platform: "win32"}, keys.sign(challenge))
	if err != nil {
		return append(checks, fail("challenge doğrulama", err.Error()))
	}
	checks = append(checks, pass("challenge imzalanarak giriş yapılıyor", "imza doğrulandı"))

	// Sahte imza reddedilmeli.
	forgedChallenge, err := h.deviceChallenge(email, fingerprint)
	if err != nil {
		return append(checks, fail("sahte imza kontrolü", err.Error()))
	}
	attacker, err := newKeyPair()
	if err != nil {
		return append(checks, fail("sahte imza kontrolü", err.Error()))
	}
	if _, err := h.loginFull(email, &deviceInput{Fingerprint: fingerprint}, attacker.sign(forgedChallenge)); err == nil {
		checks = append(checks, fail("sahte imza reddediliyor", "başka anahtarla üretilmiş imza kabul edildi"))
	} else {
		checks = append(checks, pass("sahte imza reddediliyor", err.Error()))
	}

	// İptal ve iptal sonrası token reddi.
	deviceID := devices.MyDevices[0].ID
	if signed.Device != nil {
		deviceID = signed.Device.ID
	}
	revokeResp, err := h.query(signed.AccessToken, fmt.Sprintf(`mutation { revokeDevice(deviceId:%q) }`, deviceID))
	if err != nil || len(revokeResp.Errors) > 0 {
		return append(checks, fail("cihaz iptali", prettyJSON(revokeResp.Errors)))
	}
	checks = append(checks, pass("cihaz iptal ediliyor", deviceID[:8]))

	afterResp, err := h.query(signed.AccessToken, `{ me { email } }`)
	if err != nil {
		return append(checks, fail("iptal sonrası token reddi", err.Error()))
	}
	if len(afterResp.Errors) == 0 && afterResp.status < 400 {
		checks = append(checks, fail("iptal sonrası token reddi", "iptal edilen cihazın token'ı hâlâ çalışıyor"))
	} else {
		checks = append(checks, pass("iptal edilen cihazın token'ı anında reddediliyor",
			fmt.Sprintf("HTTP %d: %s", afterResp.status, afterResp.firstError())))
	}

	return checks
}

func (h *harness) deviceChallenge(email, fingerprint string) (string, error) {
	resp, err := h.query("", fmt.Sprintf(
		`mutation { requestDeviceChallenge(input:{email:%q, fingerprint:%q}) { challenge expiresAt } }`,
		email, fingerprint))
	if err != nil {
		return "", err
	}
	if len(resp.Errors) > 0 {
		return "", fmt.Errorf("%s", resp.firstError())
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

// 12. KVKK: LLM isteğinde PII maskelenmiş, zorunlu kriter ihlali KALDI
// üretiyor, veri dışa aktarma tüm kişisel veriyi döndürüyor.
func verifyKVKK(h *harness) []check {
	var checks []check

	// PII maskeleme: sağlayıcının gördüğü metin ile transkript farklı olmalı.
	session, err := h.startSession(h.traineeToken, kvkkScenario)
	if err != nil {
		return append(checks, fail("KVKK senaryosu açılıyor", err.Error()))
	}

	message := fmt.Sprintf(
		"Kimlik numaram %s, IBAN %s, telefon 0532 123 45 67, e-posta test@example.com",
		sampleTCKN, sampleIBAN)

	turnResp, err := h.submitTurn(h.traineeToken, session, message)
	if err != nil || len(turnResp.Errors) > 0 {
		return append(checks, fail("kişisel veri içeren sıra gönderiliyor", prettyJSON(turnResp.Errors)))
	}

	prompt, err := h.lastPrompt()
	if err != nil {
		checks = append(checks, fail("LLM isteğinde PII maskelenmiş", err.Error()))
	} else {
		var leaked []string
		for _, secret := range []string{sampleTCKN, "TR330006100519786457841326", "05321234567", "test@example.com"} {
			normalized := strings.ReplaceAll(strings.ReplaceAll(prompt, " ", ""), "-", "")
			if strings.Contains(normalized, strings.ReplaceAll(secret, " ", "")) {
				leaked = append(leaked, secret)
			}
		}
		if len(leaked) > 0 {
			checks = append(checks, fail("LLM isteğinde PII maskelenmiş",
				"sağlayıcıya sızan değerler: "+strings.Join(leaked, ", ")))
		} else if !strings.Contains(prompt, "[TCKN]") || !strings.Contains(prompt, "[IBAN]") {
			checks = append(checks, fail("LLM isteğinde PII maskelenmiş",
				"maskeleme etiketleri yok: "+truncateString(prompt)))
		} else {
			checks = append(checks, pass("LLM isteğinde TC ve IBAN maskelenmiş",
				truncateString(strings.TrimSpace(prompt))))
		}
	}

	// Transkript ORİJİNAL metni tutmalı: puanlama doğruluğu buna bağlı.
	turns, err := h.sessionTurns(h.traineeToken, session)
	if err != nil {
		checks = append(checks, fail("transkriptte orijinal metin duruyor", err.Error()))
	} else {
		found := false
		masked := 0
		for _, t := range turns {
			if strings.Contains(t.Text, sampleTCKN) {
				found = true
			}
			masked += t.MaskedFieldCount
		}
		if !found {
			checks = append(checks, fail("transkriptte orijinal metin duruyor",
				"transkript de maskelenmiş; puanlama ifşayı göremez"))
		} else if masked == 0 {
			checks = append(checks, fail("maskelenen alan sayısı kaydediliyor", "sayı sıfır"))
		} else {
			checks = append(checks, pass("maskeleme yalnızca giden kopyada, sayı kayıtta",
				fmt.Sprintf("%d alan maskelendi, transkript orijinal", masked)))
		}
	}

	// Zorunlu kriter ihlali KALDI üretmeli.
	if err := h.endSession(h.traineeToken, session); err != nil {
		checks = append(checks, fail("KVKK oturumu bitiriliyor", err.Error()))
	} else {
		score, err := h.sessionScore(h.traineeToken, session)
		if err != nil {
			checks = append(checks, fail("zorunlu kriter ihlali KALDI üretiyor", err.Error()))
		} else if len(score.FailedMandatoryKeys) == 0 {
			checks = append(checks, fail("zorunlu kriter ihlali KALDI üretiyor",
				"zorunlu kriter düşmedi; senaryo tuzağı çalışmıyor olabilir"))
		} else if score.Passed {
			checks = append(checks, fail("zorunlu kriter ihlali KALDI üretiyor",
				fmt.Sprintf("zorunlu kriter düştü (%v) ama sonuç GEÇTİ", score.FailedMandatoryKeys)))
		} else {
			ratio := 0.0
			if score.MaxTotal > 0 {
				ratio = score.Total / score.MaxTotal * 100
			}
			checks = append(checks, pass("zorunlu kriter ihlali doğrudan KALDI üretiyor",
				fmt.Sprintf("puan %.0f%% olmasına rağmen KALDI, düşen kriter: %v",
					ratio, score.FailedMandatoryKeys)))
		}
	}

	// Rubrikte KVKK tuzağı tanımlı ve zorunlu işaretli olmalı.
	rubricResp, err := h.query(h.adminToken, fmt.Sprintf(
		`{ rubric(lineageId:%q) { criteria { key mandatory trap } } }`, serviceRubric))
	if err != nil {
		checks = append(checks, fail("KVKK tuzağı rubrikte tanımlı", err.Error()))
	} else {
		var rubric struct {
			Rubric struct {
				Criteria []struct {
					Key       string  `json:"key"`
					Mandatory bool    `json:"mandatory"`
					Trap      *string `json:"trap"`
				} `json:"criteria"`
			} `json:"rubric"`
		}
		if err := rubricResp.decode(&rubric); err != nil {
			checks = append(checks, fail("KVKK tuzağı rubrikte tanımlı", err.Error()))
		} else {
			trapped := false
			for _, c := range rubric.Rubric.Criteria {
				if c.Trap != nil && *c.Trap != "" && c.Mandatory {
					trapped = true
				}
			}
			if !trapped {
				checks = append(checks, fail("KVKK tuzağı zorunlu kriterde tanımlı",
					"tuzaklı ve zorunlu kriter yok"))
			} else {
				checks = append(checks, pass("KVKK tuzağı zorunlu kriterde tanımlı",
					"tohum rubriğinde tuzak + mandatory"))
			}
		}
	}

	// Dışa aktarma tüm kişisel veriyi döndürmeli.
	exportResp, err := h.query(h.traineeToken, `mutation { exportMyData {
		user { email } devices { name } sessions { id } auditLog { action } document } }`)
	if err != nil || len(exportResp.Errors) > 0 {
		checks = append(checks, fail("veri dışa aktarma tüm kişisel veriyi döndürüyor",
			prettyJSON(exportResp.Errors)))
	} else {
		var export struct {
			ExportMyData struct {
				User struct {
					Email string `json:"email"`
				} `json:"user"`
				Sessions []struct{}     `json:"sessions"`
				AuditLog []struct{}     `json:"auditLog"`
				Document map[string]any `json:"document"`
			} `json:"exportMyData"`
		}
		if err := exportResp.decode(&export); err != nil {
			checks = append(checks, fail("veri dışa aktarma", err.Error()))
		} else {
			doc := export.ExportMyData.Document
			_, hasProfile := doc["profile"]
			sessions, hasSessions := doc["sessions"].([]any)
			hasTranscript := false
			if hasSessions && len(sessions) > 0 {
				if first, ok := sessions[0].(map[string]any); ok {
					_, hasTranscript = first["transcript"]
				}
			}
			switch {
			case !hasProfile:
				checks = append(checks, fail("dışa aktarma profil içeriyor", "profile alanı yok"))
			case !hasSessions || len(sessions) == 0:
				checks = append(checks, fail("dışa aktarma oturumları içeriyor", "sessions boş"))
			case !hasTranscript:
				checks = append(checks, fail("dışa aktarma transkript içeriyor", "transcript alanı yok"))
			case len(export.ExportMyData.AuditLog) == 0:
				checks = append(checks, fail("dışa aktarma denetim kaydı içeriyor", "auditLog boş"))
			default:
				checks = append(checks, pass("dışa aktarma profil, oturum, transkript ve denetimi içeriyor",
					fmt.Sprintf("%d oturum, %d denetim kaydı",
						len(sessions), len(export.ExportMyData.AuditLog))))
			}
		}
	}

	// Ham ses saklama varsayılanı kapalı olmalı.
	checks = append(checks, checkRawAudioDefault())

	return checks
}

// checkRawAudioDefault, ham ses saklamanın varsayılan olarak kapalı
// olduğunu ve belgelendiğini doğrular.
func checkRawAudioDefault() check {
	out, err := runGo("run", "./cmd/configdump")
	if err != nil {
		return fail("ham ses saklama varsayılanı kapalı", strings.TrimSpace(out))
	}
	if !strings.Contains(out, "store_raw_audio=false") {
		return fail("ham ses saklama varsayılanı kapalı", strings.TrimSpace(out))
	}
	return pass("ham ses saklama varsayılanı kapalı", "LLM_STORE_RAW_AUDIO=false")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
