package main

import (
	"fmt"
	"strings"
)

// 1. GraphQL: sorgu, mutation ve subscription çalışıyor; izin korumalı alan
// yetkisiz kullanıcıda null; eski REST auth uçları 404.
func verifyGraphQL(h *harness) []check {
	var checks []check

	// Sorgu
	resp, err := h.query(h.traineeToken, `{ me { id email organizationId permissions } }`)
	if err != nil {
		return append(checks, fail("me sorgusu", err.Error()))
	}
	var me struct {
		Me struct {
			ID             string   `json:"id"`
			Email          string   `json:"email"`
			OrganizationID string   `json:"organizationId"`
			Permissions    []string `json:"permissions"`
		} `json:"me"`
	}
	if err := resp.decode(&me); err != nil {
		checks = append(checks, fail("me sorgusu", err.Error()))
	} else if me.Me.Email != traineeEmail {
		checks = append(checks, fail("me sorgusu", "beklenmeyen kullanıcı: "+me.Me.Email))
	} else if me.Me.OrganizationID == "00000000-0000-0000-0000-000000000000" {
		checks = append(checks, fail("me sorgusu", "organizasyon kimliği sıfır UUID"))
	} else {
		checks = append(checks, pass("query çalışıyor",
			fmt.Sprintf("me → %s, org %s", me.Me.Email, me.Me.OrganizationID[:8])))
	}

	// Mutation
	if _, err := h.mustQuery(h.traineeToken,
		`mutation { updateProfile(input:{firstName:"Demo"}) { firstName } }`); err != nil {
		checks = append(checks, fail("mutation çalışıyor", err.Error()))
	} else {
		checks = append(checks, pass("mutation çalışıyor", "updateProfile"))
	}

	// Subscription: şemada tanımlı ve WebSocket transport'u bağlı.
	checks = append(checks, h.checkSubscription())

	// İzin korumalı alan: nullable alanda null, sorgunun tamamı patlamıyor.
	checks = append(checks, h.checkOverrideNull())

	// Eski REST auth uçları
	restPaths := []string{"/api/v1/auth/login/request", "/api/v1/auth/login/verify", "/api/v1/me", "/api/v1/users"}
	allGone := true
	for _, path := range restPaths {
		status, err := h.restStatus("POST", path)
		if err != nil || status != 404 {
			checks = append(checks, fail("eski REST uçları 404",
				fmt.Sprintf("%s → %d (%v)", path, status, err)))
			allGone = false
			break
		}
	}
	if allGone {
		checks = append(checks, pass("eski REST auth uçları 404", fmt.Sprintf("%d uç denendi", len(restPaths))))
	}

	// Sağlık uçları REST olarak kalmalı.
	if status, err := h.restStatus("GET", "/health/live"); err != nil || status != 200 {
		checks = append(checks, fail("sağlık ucu çalışıyor", fmt.Sprintf("HTTP %d (%v)", status, err)))
	} else {
		checks = append(checks, pass("sağlık ucu REST olarak kaldı", "/health/live → 200"))
	}

	return checks
}

// checkSubscription, abonelik kanalının bağlandığını doğrular.
//
// WebSocket el sıkışması gerçekten kuruluyor ve connectionParams içindeki
// token doğrulanıyor. Şemada alanın var olduğuna bakmak yetmezdi: transport
// bağlanmamış olsaydı şema yine geçerli görünürdü.
func (h *harness) checkSubscription() check {
	session, err := h.startSession(h.traineeToken, refundScenario)
	if err != nil {
		return fail("subscription çalışıyor", "oturum açılamadı: "+err.Error())
	}

	events, err := h.subscribeSessionEvents(h.traineeToken, session)
	if err != nil {
		return fail("subscription çalışıyor", err.Error())
	}
	return pass("subscription çalışıyor", fmt.Sprintf("%d olay alındı", events))
}

// checkOverrideNull, ezme alanının yetkiye göre davranışını doğrular.
func (h *harness) checkOverrideNull() check {
	session, score, err := h.playAndScore(h.traineeToken, refundScenario,
		"Merhaba, iade talebiniz için sipariş numaranızı alabilir miyim?")
	if err != nil {
		return fail("izin korumalı alan", err.Error())
	}

	// Yetkili kullanıcı ezer. mustQuery kullanılıyor: query() GraphQL
	// hatasını yanıtın içinde bırakıyor ve incelenmezse uygulanmamış bir
	// mutation başarılı sayılır.
	if _, err := h.mustQuery(h.adminToken, fmt.Sprintf(
		`mutation { overrideScore(input:{scoreId:%q, total:20, passed:true, reason:"doğrulama koşusu"}) { total passed override { reason } } }`,
		score)); err != nil {
		return fail("izin korumalı alan", "ezme başarısız: "+err.Error())
	}

	// Çalışan: ezme alanı null, ama puanın kendisi görünüyor.
	resp, err := h.query(h.traineeToken, fmt.Sprintf(
		`{ session(id:%q) { score { total passed override { reason } } } }`, session))
	if err != nil {
		return fail("izin korumalı alan", err.Error())
	}

	var traineeView struct {
		Session struct {
			Score struct {
				Total    float64 `json:"total"`
				Override *struct {
					Reason string `json:"reason"`
				} `json:"override"`
			} `json:"score"`
		} `json:"session"`
	}
	if err := resp.decode(&traineeView); err != nil {
		return fail("izin korumalı alan", err.Error())
	}
	if traineeView.Session.Score.Override != nil {
		return fail("izin korumalı alan", "yetkisiz kullanıcıda ezme alanı null olmalıydı")
	}
	if traineeView.Session.Score.Total == 0 {
		return fail("izin korumalı alan", "yetkisiz kullanıcı puanı görebilmeliydi")
	}

	// Yönetici: ezme alanı dolu.
	adminResp, err := h.query(h.adminToken, fmt.Sprintf(
		`{ session(id:%q) { score { override { reason previousTotal } } } }`, session))
	if err != nil {
		return fail("izin korumalı alan", err.Error())
	}
	if !strings.Contains(string(adminResp.Data), "doğrulama") {
		return fail("izin korumalı alan", "yetkili kullanıcıda ezme alanı dolu gelmeliydi: "+string(adminResp.Data))
	}

	return pass("izin korumalı alan",
		"ezme çalışanda null, yöneticide dolu; puan her ikisinde de görünüyor")
}

// 3. Web arayüzü: şema SDL olarak dışa aktarılmış, CORS izinli origin çalışıyor.
func verifyWeb(h *harness) []check {
	var checks []check

	checks = append(checks, checkSchemaExport())
	checks = append(checks, h.checkCORS())

	return checks
}

// 4. Electron: cihaz uçları çalışıyor, üç platform normalize ediliyor.
func verifyElectron(h *harness) []check {
	var checks []check

	platforms := map[string]string{
		"win32":  "WINDOWS",
		"darwin": "MACOS",
		"linux":  "LINUX",
	}

	for raw, want := range platforms {
		email := h.newEmail("platform")
		device := &deviceInput{
			Fingerprint: "fp-" + randomHex(8) + "-electron",
			Name:        "Electron " + raw,
			Platform:    raw,
		}
		result, err := h.loginFull(email, device, "")
		if err != nil {
			checks = append(checks, fail("platform normalizasyonu ("+raw+")", err.Error()))
			continue
		}
		if result.Device == nil {
			checks = append(checks, fail("platform normalizasyonu ("+raw+")", "cihaz eşleşmedi"))
			continue
		}
		if result.Device.Platform != want {
			checks = append(checks, fail("platform normalizasyonu ("+raw+")",
				fmt.Sprintf("%s bekleniyordu, %s geldi", want, result.Device.Platform)))
			continue
		}
		checks = append(checks, pass("platform normalizasyonu", raw+" → "+want))
	}

	// Cihaz listeleme ve iptal uçları.
	email := h.newEmail("cihaz-uc")
	device := &deviceInput{Fingerprint: "fp-" + randomHex(8) + "-list", Name: "Liste", Platform: "linux"}
	result, err := h.loginFull(email, device, "")
	if err != nil {
		return append(checks, fail("cihaz uçları", err.Error()))
	}

	resp, err := h.query(result.AccessToken, `{ myDevices { id name platform hasPublicKey } }`)
	if err != nil {
		return append(checks, fail("cihaz listeleme", err.Error()))
	}
	var list struct {
		MyDevices []struct {
			ID string `json:"id"`
		} `json:"myDevices"`
	}
	if err := resp.decode(&list); err != nil || len(list.MyDevices) == 0 {
		return append(checks, fail("cihaz listeleme", "cihaz listesi boş: "+prettyJSON(resp.Errors)))
	}
	checks = append(checks, pass("cihaz listeleme", fmt.Sprintf("%d cihaz", len(list.MyDevices))))

	revokeResp, err := h.query(result.AccessToken, fmt.Sprintf(
		`mutation { revokeDevice(deviceId:%q) }`, list.MyDevices[0].ID))
	if err != nil || len(revokeResp.Errors) > 0 {
		return append(checks, fail("cihaz iptali", prettyJSON(revokeResp.Errors)))
	}
	checks = append(checks, pass("cihaz iptali", "revokeDevice → true"))

	return checks
}
