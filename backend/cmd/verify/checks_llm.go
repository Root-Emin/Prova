package main

import (
	"fmt"
	"strings"
	"time"
)

// 5. İki LLM: profillerde iki kademe kayıtlı, konuşma hızlıyı, puanlama
// güçlüyü kullanıyor.
func verifyTwoTiers(h *harness) []check {
	var checks []check

	resp, err := h.query(h.adminToken, `{ llmProfiles { tier model status version } }`)
	if err != nil {
		return append(checks, fail("iki kademe kayıtlı", err.Error()))
	}
	var payload struct {
		LLMProfiles []struct {
			Tier   string `json:"tier"`
			Model  string `json:"model"`
			Status string `json:"status"`
		} `json:"llmProfiles"`
	}
	if err := resp.decode(&payload); err != nil {
		return append(checks, fail("iki kademe kayıtlı", err.Error()))
	}

	// Liste soy başına SON sürümü döndürüyor; yönetici bir profili
	// düzenlediyse o sürüm taslak olur. Yürürlükteki sürüm ayrı bir soru,
	// ve cevabı yayınlanmış belgelerde: listeye bakıp "yayınlanmış yok"
	// demek, çalışan bir kurulumu yanlışlıkla FAIL yapardı.
	tiers := map[string]string{}
	for _, p := range payload.LLMProfiles {
		tiers[p.Tier] = p.Model
	}
	if tiers["FAST"] == "" || tiers["STRONG"] == "" {
		checks = append(checks, fail("iki kademe kayıtlı",
			fmt.Sprintf("profiller: %v", tiers)))
	} else {
		checks = append(checks, pass("iki kademe kayıtlı",
			fmt.Sprintf("hızlı=%s, güçlü=%s", tiers["FAST"], tiers["STRONG"])))
	}

	// Her kademede YÜRÜRLÜKTE (yayınlanmış) bir profil olmalı: çalışma
	// zamanı yalnızca yayınlanmışları okuyor.
	for _, tier := range []string{"fast", "strong"} {
		count, err := mongo(fmt.Sprintf(
			`db.llm_profiles.countDocuments({tier: "%s", status: "published"})`, tier))
		if err != nil {
			checks = append(checks, fail("yayınlanmış profil ("+tier+")", err.Error()))
			continue
		}
		if strings.TrimSpace(count) == "0" {
			checks = append(checks, fail("yayınlanmış profil ("+tier+")",
				"bu kademede yayınlanmış profil yok"))
			continue
		}
		checks = append(checks, pass("yayınlanmış profil var ("+tier+")",
			strings.TrimSpace(count)+" sürüm"))
	}

	// Bir oturum oyna ve hangi kademelerin kullanıldığına bak.
	session, _, err := h.playAndScore(h.traineeToken, refundScenario,
		"Merhaba, iade politikamızı açıklayayım.")
	if err != nil {
		return append(checks, fail("kademe kullanımı", err.Error()))
	}

	records, err := h.routingRecords(session)
	if err != nil {
		return append(checks, fail("kademe kullanımı", err.Error()))
	}

	var turnTier, scoringTier string
	for _, r := range records {
		if r.Rule == "scoring" {
			scoringTier = r.Tier
		}
		if r.Rule == "default" && turnTier == "" {
			turnTier = r.Tier
		}
	}

	if turnTier != "FAST" {
		checks = append(checks, fail("konuşma hızlı kademeyi kullanıyor",
			fmt.Sprintf("kademe: %q, kayıtlar: %s", turnTier, prettyJSON(records))))
	} else {
		checks = append(checks, pass("konuşma hızlı kademeyi kullanıyor", "rule=default → FAST"))
	}

	if scoringTier != "STRONG" {
		checks = append(checks, fail("puanlama güçlü kademeyi kullanıyor",
			fmt.Sprintf("kademe: %q", scoringTier)))
	} else {
		checks = append(checks, pass("puanlama güçlü kademeyi kullanıyor", "rule=scoring → STRONG"))
	}

	// Puan alıntılı ve alıntılar doğrulanmış olmalı.
	score, err := h.sessionScore(h.traineeToken, session)
	if err != nil {
		return append(checks, fail("alıntılı puan üretiliyor", err.Error()))
	}
	verified, total := 0, len(score.Criteria)
	for _, c := range score.Criteria {
		if c.QuoteVerified {
			verified++
		}
	}
	if total == 0 {
		checks = append(checks, fail("alıntılı puan üretiliyor", "kriter yok"))
	} else if verified == 0 {
		checks = append(checks, fail("alıntılı puan üretiliyor",
			"hiçbir alıntı doğrulanmadı; alıntı doğrulaması çalışmıyor olabilir"))
	} else {
		checks = append(checks, pass("alıntılı puan üretiliyor ve alıntılar doğrulanıyor",
			fmt.Sprintf("%d/%d alıntı transkriptte bulundu", verified, total)))
	}

	return checks
}

// routingRecordView, doğrulamanın ilgilendiği yönlendirme alanları.
type routingRecordView struct {
	Tier    string  `json:"tier"`
	Rule    string  `json:"rule"`
	Model   string  `json:"model"`
	Success bool    `json:"success"`
	CostUSD float64 `json:"costUsd"`
}

func (h *harness) routingRecords(sessionID string) ([]routingRecordView, error) {
	resp, err := h.query(h.adminToken, fmt.Sprintf(
		`{ routingRecords(sessionId:%q, limit:50) { tier rule model success costUsd } }`, sessionID))
	if err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("%s", resp.firstError())
	}

	var payload struct {
		RoutingRecords []routingRecordView `json:"routingRecords"`
	}
	if err := resp.decode(&payload); err != nil {
		return nil, err
	}
	return payload.RoutingRecords, nil
}

// 6. Dağılım: bir oturumun kayıtlarında iki kademe de var, failover kaydı
// üretilebiliyor, istatistik tasarruf yüzdesi döndürüyor.
func verifyRouting(h *harness) []check {
	var checks []check

	session, err := h.startSession(h.traineeToken, refundScenario)
	if err != nil {
		return append(checks, fail("oturum açılıyor", err.Error()))
	}

	// Olağan sıra: hızlı kademe.
	if resp, err := h.submitTurn(h.traineeToken, session, "Merhaba, nasıl yardımcı olabilirim?"); err != nil || len(resp.Errors) > 0 {
		return append(checks, fail("olağan konuşma sırası", prettyJSON(resp.Errors)))
	}

	// Hızlı sağlayıcıyı bilerek boz.
	if err := h.breakModel(fastModelName); err != nil {
		return append(checks, fail("hızlı sağlayıcı bozuluyor", err.Error()))
	}
	// Kontrol ne olursa olsun sağlayıcıyı onar: bozuk bırakmak sonraki
	// gereksinimlerin hepsini yanlış nedenle düşürürdü.
	defer func() { _ = h.healModel(fastModelName) }()

	turnResp, err := h.submitTurn(h.traineeToken, session, "Ürünü kullandınız mı acaba?")
	if err != nil || len(turnResp.Errors) > 0 {
		checks = append(checks, fail("failover sırasında oturum kesintisiz devam ediyor",
			fmt.Sprintf("%v %s", err, prettyJSON(turnResp.Errors))))
	} else {
		checks = append(checks, pass("failover sırasında oturum kesintisiz devam ediyor",
			"hızlı sağlayıcı kapalıyken konuşma sırası tamamlandı"))
	}

	if err := h.healModel(fastModelName); err != nil {
		checks = append(checks, fail("sağlayıcı onarılıyor", err.Error()))
	}

	if err := h.endSession(h.traineeToken, session); err != nil {
		checks = append(checks, fail("oturum bitiriliyor", err.Error()))
	}

	records, err := h.routingRecords(session)
	if err != nil {
		return append(checks, fail("yönlendirme kayıtları", err.Error()))
	}

	seenTiers := map[string]bool{}
	failover := false
	for _, r := range records {
		seenTiers[r.Tier] = true
		if r.Rule == "failover" {
			failover = true
		}
	}
	if !seenTiers["FAST"] || !seenTiers["STRONG"] {
		checks = append(checks, fail("kayıtlarda iki kademe de var",
			fmt.Sprintf("görülen kademeler: %v", seenTiers)))
	} else {
		checks = append(checks, pass("bir oturumun kayıtlarında iki kademe de var",
			fmt.Sprintf("%d kayıt", len(records))))
	}
	if !failover {
		checks = append(checks, fail("failover kaydı üretiliyor",
			"rule=failover kaydı yok: "+prettyJSON(records)))
	} else {
		checks = append(checks, pass("failover kaydı üretiliyor", "rule=failover kaydedildi"))
	}

	// İstatistik: grafik çizilebilir veri.
	statsResp, err := h.query(h.adminToken, `{ routingStats {
		perTier { tier calls avgLatencyMs costUsd inputTokens outputTokens }
		failoverCount totalCostUsd allStrongCostUsd savingsPercent
	} }`)
	if err != nil {
		return append(checks, fail("istatistik sorgusu", err.Error()))
	}
	var stats struct {
		RoutingStats struct {
			PerTier []struct {
				Tier  string  `json:"tier"`
				Calls int     `json:"calls"`
				Cost  float64 `json:"costUsd"`
			} `json:"perTier"`
			FailoverCount    int     `json:"failoverCount"`
			TotalCostUsd     float64 `json:"totalCostUsd"`
			AllStrongCostUsd float64 `json:"allStrongCostUsd"`
			SavingsPercent   float64 `json:"savingsPercent"`
		} `json:"routingStats"`
	}
	if err := statsResp.decode(&stats); err != nil {
		return append(checks, fail("istatistik sorgusu", err.Error()))
	}

	s := stats.RoutingStats
	switch {
	case len(s.PerTier) < 2:
		checks = append(checks, fail("istatistik kademe başına veri döndürüyor",
			fmt.Sprintf("%d kademe", len(s.PerTier))))
	case s.FailoverCount == 0:
		checks = append(checks, fail("istatistik failover sayıyor", "failoverCount = 0"))
	case s.AllStrongCostUsd <= 0:
		checks = append(checks, fail("istatistik karşılaştırma maliyeti üretiyor", "allStrongCostUsd = 0"))
	case s.SavingsPercent <= 0:
		checks = append(checks, fail("istatistik tasarruf yüzdesi döndürüyor",
			fmt.Sprintf("savingsPercent = %.2f", s.SavingsPercent)))
	default:
		checks = append(checks, pass("istatistik tasarruf yüzdesi döndürüyor",
			fmt.Sprintf("%.1f%% tasarruf (%.6f$ / %.6f$), %d failover, %d kademe",
				s.SavingsPercent, s.TotalCostUsd, s.AllStrongCostUsd, s.FailoverCount, len(s.PerTier))))
	}

	return checks
}

// 11. LLM manipülasyonu: karakter değişimi yeni sürüm üretiyor, profil
// değişimi denetime düşüyor, test mutation'ı yanıt veriyor.
func verifyLLMAdmin(h *harness) []check {
	var checks []check

	// Karakter değişimi yeni sürüm üretiyor (ve zorluk gerçekten değişiyor).
	created, err := h.createCharacter("LLM yönetimi "+h.runID, "LOW")
	if err != nil {
		return append(checks, fail("karakter değişimi yeni sürüm üretiyor", err.Error()))
	}
	if err := h.publishCharacter(created.LineageID, created.Version); err != nil {
		return append(checks, fail("karakter yayınlama", err.Error()))
	}
	updated, err := h.updateCharacter(created.LineageID, "LLM yönetimi (zorlu)", "EXTREME")
	if err != nil {
		return append(checks, fail("karakter değişimi yeni sürüm üretiyor", err.Error()))
	}
	if updated.Version <= created.Version || updated.Difficulty != "EXTREME" {
		checks = append(checks, fail("karakter değişimi yeni sürüm üretiyor",
			fmt.Sprintf("v%d/%s", updated.Version, updated.Difficulty)))
	} else {
		checks = append(checks, pass("karakter değişimi yeni sürüm üretiyor",
			fmt.Sprintf("v%d LOW → v%d EXTREME", created.Version, updated.Version)))
	}

	// Profil değişimi denetime düşüyor.
	before, err := h.countAudit("llm.profile.changed")
	if err != nil {
		return append(checks, fail("profil değişimi denetime düşüyor", err.Error()))
	}

	profileResp, err := h.query(h.adminToken, fmt.Sprintf(
		`mutation { updateLLMProfile(lineageId:%q, input:{
			tier:FAST, provider:"openai-compatible", baseUrl:%q, model:%q,
			temperature:0.55, topP:0.9, maxTokens:700,
			systemPromptSuffix:"Doğrulama koşusu %s",
			inputCostPer1K:0.00015, outputCostPer1K:0.0006
		}) { version status temperature systemPromptSuffix } }`,
		fastProfile, h.mockLLM+"/v1", fastModelName, h.runID))
	if err != nil || len(profileResp.Errors) > 0 {
		return append(checks, fail("profil çalışma anında değiştirilebiliyor",
			fmt.Sprintf("%v %s", err, prettyJSON(profileResp.Errors))))
	}
	var profilePayload struct {
		UpdateLLMProfile struct {
			Version     int     `json:"version"`
			Temperature float64 `json:"temperature"`
		} `json:"updateLLMProfile"`
	}
	if err := profileResp.decode(&profilePayload); err != nil {
		return append(checks, fail("profil çalışma anında değiştirilebiliyor", err.Error()))
	}
	checks = append(checks, pass("profil çalışma anında değiştirilebiliyor",
		fmt.Sprintf("v%d, sıcaklık %.2f", profilePayload.UpdateLLMProfile.Version,
			profilePayload.UpdateLLMProfile.Temperature)))

	// Denetim yazımı eşzamansız; kısa bir tolerans tanınıyor.
	after := before
	for i := 0; i < 20 && after <= before; i++ {
		time.Sleep(100 * time.Millisecond)
		after, _ = h.countAudit("llm.profile.changed")
	}
	if after <= before {
		checks = append(checks, fail("profil değişimi denetime düşüyor",
			fmt.Sprintf("kayıt sayısı artmadı: %d", after)))
	} else {
		checks = append(checks, pass("profil değişimi denetime düşüyor",
			fmt.Sprintf("llm.profile.changed: %d → %d", before, after)))
	}

	// Test mutation'ı yanıt veriyor.
	probeResp, err := h.query(h.adminToken, fmt.Sprintf(
		`mutation { testLLMProfile(input:{profileLineageId:%q, prompt:"Merhaba, kimsin?"}) {
			model output latencyMs inputTokens costUsd error } }`, fastProfile))
	if err != nil {
		return append(checks, fail("test mutation'ı yanıt veriyor", err.Error()))
	}
	var probe struct {
		TestLLMProfile struct {
			Model     string  `json:"model"`
			Output    string  `json:"output"`
			LatencyMs int     `json:"latencyMs"`
			Error     *string `json:"error"`
		} `json:"testLLMProfile"`
	}
	if err := probeResp.decode(&probe); err != nil {
		return append(checks, fail("test mutation'ı yanıt veriyor", err.Error()))
	}
	if probe.TestLLMProfile.Error != nil {
		checks = append(checks, fail("test mutation'ı yanıt veriyor",
			"sağlayıcı hatası: "+*probe.TestLLMProfile.Error))
	} else if strings.TrimSpace(probe.TestLLMProfile.Output) == "" {
		checks = append(checks, fail("test mutation'ı yanıt veriyor", "boş yanıt"))
	} else {
		checks = append(checks, pass("test mutation'ı yanıt veriyor",
			fmt.Sprintf("model=%s, %dms, %d karakter",
				probe.TestLLMProfile.Model, probe.TestLLMProfile.LatencyMs,
				len(probe.TestLLMProfile.Output))))
	}

	return checks
}

// countAudit, belirli bir eylemin denetim kaydı sayısını döndürür.
func (h *harness) countAudit(action string) (int, error) {
	out, err := psql(fmt.Sprintf(`SELECT count(*) FROM audit_logs WHERE action='%s'`, action))
	if err != nil {
		return 0, err
	}
	var count int
	if _, err := fmt.Sscanf(out, "%d", &count); err != nil {
		return 0, fmt.Errorf("sayı çözümlenemedi: %q", out)
	}
	return count, nil
}
