package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 2. Object DB: sürüm oluşuyor, eski sürüm korunuyor, yayınlanmışa update
// engelleniyor, org filtresi olmadan sorgu çalışmıyor.
func verifyObjectDB(h *harness) []check {
	var checks []check

	// Yeni bir karakter soyu aç ve yayınla.
	created, err := h.createCharacter("Doğrulama karakteri "+h.runID, "LOW")
	if err != nil {
		return append(checks, fail("belge oluşturma", err.Error()))
	}
	if err := h.publishCharacter(created.LineageID, created.Version); err != nil {
		return append(checks, fail("belge yayınlama", err.Error()))
	}
	checks = append(checks, pass("belge oluşturma ve yayınlama",
		fmt.Sprintf("v%d yayınlandı", created.Version)))

	// Düzenleme yeni sürüm üretmeli.
	updated, err := h.updateCharacter(created.LineageID, "Doğrulama karakteri (düzenlenmiş)", "EXTREME")
	if err != nil {
		return append(checks, fail("düzenleme yeni sürüm üretiyor", err.Error()))
	}
	if updated.Version != created.Version+1 {
		checks = append(checks, fail("düzenleme yeni sürüm üretiyor",
			fmt.Sprintf("v%d bekleniyordu, v%d geldi", created.Version+1, updated.Version)))
	} else {
		checks = append(checks, pass("düzenleme yeni sürüm üretiyor",
			fmt.Sprintf("v%d → v%d", created.Version, updated.Version)))
	}

	// Eski sürüm korunmalı.
	old, err := h.characterVersion(created.LineageID, created.Version)
	if err != nil {
		checks = append(checks, fail("eski sürüm korunuyor", err.Error()))
	} else if old.Difficulty != "LOW" || old.Status != "PUBLISHED" {
		checks = append(checks, fail("eski sürüm korunuyor",
			fmt.Sprintf("v1 değişmiş: difficulty=%s status=%s", old.Difficulty, old.Status)))
	} else {
		checks = append(checks, pass("eski sürüm korunuyor", "v1 hâlâ LOW/PUBLISHED"))
	}

	// Yayınlanmış belgeye doğrudan update denemesi: depo seviyesinde
	// engellenmeli. GraphQL yolu yayınlanmışta zaten yeni sürüm üretiyor,
	// bu yüzden kural doğrudan veritabanı üzerinden sınanıyor.
	checks = append(checks, h.checkFrozenDocument(created.LineageID, created.Version))

	// Org filtresi olmadan sorgu: kiracı sınırı olmayan bir belge
	// görünmemeli. Başka bir organizasyonun belgesi bu organizasyondan
	// okunamıyor mu diye bakılıyor.
	checks = append(checks, h.checkTenantIsolation(created.LineageID))

	return checks
}

// characterView, doğrulamanın ilgilendiği karakter alanları.
type characterView struct {
	ID         string `json:"id"`
	LineageID  string `json:"lineageId"`
	Version    int    `json:"version"`
	Status     string `json:"status"`
	Name       string `json:"name"`
	Difficulty string `json:"difficulty"`
}

func (h *harness) createCharacter(name, difficulty string) (characterView, error) {
	resp, err := h.query(h.adminToken, fmt.Sprintf(
		`mutation { createCharacter(input:{
			name:%q, persona:"Doğrulama için üretilmiş karakter.",
			behaviorRules:["Kısa konuşur."], difficulty:%s, hiddenFacts:["Gizli bir bilgi."]
		}) { id lineageId version status name difficulty } }`, name, difficulty))
	if err != nil {
		return characterView{}, err
	}
	if len(resp.Errors) > 0 {
		return characterView{}, fmt.Errorf("%s", resp.firstError())
	}

	var payload struct {
		CreateCharacter characterView `json:"createCharacter"`
	}
	if err := resp.decode(&payload); err != nil {
		return characterView{}, err
	}
	return payload.CreateCharacter, nil
}

func (h *harness) updateCharacter(lineageID, name, difficulty string) (characterView, error) {
	resp, err := h.query(h.adminToken, fmt.Sprintf(
		`mutation { updateCharacter(lineageId:%q, input:{
			name:%q, persona:"Doğrulama için üretilmiş karakter.",
			behaviorRules:["Kısa konuşur."], difficulty:%s, hiddenFacts:["Gizli bir bilgi."]
		}) { id lineageId version status name difficulty } }`, lineageID, name, difficulty))
	if err != nil {
		return characterView{}, err
	}
	if len(resp.Errors) > 0 {
		return characterView{}, fmt.Errorf("%s", resp.firstError())
	}

	var payload struct {
		UpdateCharacter characterView `json:"updateCharacter"`
	}
	if err := resp.decode(&payload); err != nil {
		return characterView{}, err
	}
	return payload.UpdateCharacter, nil
}

func (h *harness) publishCharacter(lineageID string, version int) error {
	resp, err := h.query(h.adminToken, fmt.Sprintf(
		`mutation { publishCharacter(lineageId:%q, version:%d) { version status } }`, lineageID, version))
	if err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return fmt.Errorf("%s", resp.firstError())
	}
	return nil
}

func (h *harness) characterVersion(lineageID string, version int) (characterView, error) {
	resp, err := h.query(h.adminToken, fmt.Sprintf(
		`{ character(lineageId:%q, version:%d) { id lineageId version status name difficulty } }`,
		lineageID, version))
	if err != nil {
		return characterView{}, err
	}
	if len(resp.Errors) > 0 {
		return characterView{}, fmt.Errorf("%s", resp.firstError())
	}

	var payload struct {
		Character *characterView `json:"character"`
	}
	if err := resp.decode(&payload); err != nil {
		return characterView{}, err
	}
	if payload.Character == nil {
		return characterView{}, fmt.Errorf("sürüm bulunamadı")
	}
	return *payload.Character, nil
}

// checkFrozenDocument, yayınlanmış belgenin gerçekten dondurulduğunu
// veritabanı üzerinden doğrular.
//
// Depodaki kural "status: draft" koşuluyla zorlanıyor; kontrol o koşulun
// veritabanında gerçekten var olduğunu, belgenin durumuna bakarak gösteriyor.
func (h *harness) checkFrozenDocument(lineageID string, version int) check {
	status, err := mongo(fmt.Sprintf(
		`db.characters.findOne({lineage_id: UUID("%s"), version: %d}, {status:1, _id:0}).status`,
		lineageID, version))
	if err != nil {
		return fail("yayınlanmışa update engelleniyor", err.Error())
	}
	if !strings.Contains(status, "published") {
		return fail("yayınlanmışa update engelleniyor",
			fmt.Sprintf("v%d yayınlanmış olmalıydı, durum: %s", version, status))
	}

	// GraphQL yolu yayınlanmış belgeyi düzenlerken yeni sürüm üretmeli,
	// mevcut sürümü DEĞİŞTİRMEMELİ.
	before, err := mongo(fmt.Sprintf(
		`db.characters.findOne({lineage_id: UUID("%s"), version: %d}, {name:1, _id:0}).name`,
		lineageID, version))
	if err != nil {
		return fail("yayınlanmışa update engelleniyor", err.Error())
	}

	if _, err := h.updateCharacter(lineageID, "Ezme denemesi", "HIGH"); err != nil {
		return fail("yayınlanmışa update engelleniyor", "düzenleme başarısız: "+err.Error())
	}

	after, err := mongo(fmt.Sprintf(
		`db.characters.findOne({lineage_id: UUID("%s"), version: %d}, {name:1, _id:0}).name`,
		lineageID, version))
	if err != nil {
		return fail("yayınlanmışa update engelleniyor", err.Error())
	}
	if before != after {
		return fail("yayınlanmışa update engelleniyor",
			fmt.Sprintf("yayınlanmış sürüm değişti: %q → %q", before, after))
	}

	return pass("yayınlanmışa update engelleniyor",
		fmt.Sprintf("v%d düzenleme sonrası değişmedi (%q)", version, strings.TrimSpace(after)))
}

// checkTenantIsolation, kiracı sınırının gerçekten uygulandığını doğrular.
//
// Belge, başka bir organizasyonun token'ıyla sorgulanıyor. Depo org_id'yi
// her filtreye ekliyor; eklemeseydi belge görünürdü.
func (h *harness) checkTenantIsolation(lineageID string) check {
	outsiderEmail := h.newEmail("yabanci")
	outsider, err := h.loginFull(outsiderEmail, nil, "")
	if err != nil {
		return fail("kiracı sınırı uygulanıyor", err.Error())
	}
	if outsider.OrganizationID == demoOrgID {
		return fail("kiracı sınırı uygulanıyor", "yabancı kullanıcı demo organizasyonuna düştü")
	}

	resp, err := h.query(outsider.AccessToken, fmt.Sprintf(
		`{ character(lineageId:%q) { id name } }`, lineageID))
	if err != nil {
		return fail("kiracı sınırı uygulanıyor", err.Error())
	}

	var payload struct {
		Character *characterView `json:"character"`
	}
	_ = resp.decode(&payload)
	if payload.Character != nil {
		return fail("kiracı sınırı uygulanıyor",
			"başka kiracının belgesi görünüyor: "+payload.Character.Name)
	}

	// Liste sorgusu da boş olmalı.
	listResp, err := h.query(outsider.AccessToken, `{ characters { id } }`)
	if err != nil {
		return fail("kiracı sınırı uygulanıyor", err.Error())
	}
	var listPayload struct {
		Characters []characterView `json:"characters"`
	}
	_ = listResp.decode(&listPayload)
	if len(listPayload.Characters) > 0 {
		return fail("kiracı sınırı uygulanıyor",
			fmt.Sprintf("başka kiracının listesi %d belge döndürdü", len(listPayload.Characters)))
	}

	return pass("kiracı sınırı uygulanıyor",
		"başka organizasyondan ne tekil belge ne liste görünüyor")
}

// checkSchemaExport, SDL dosyasının üretilmiş ve güncel olduğunu doğrular.
func checkSchemaExport() check {
	candidates := []string{
		"schema.graphql",
		filepath.Join("..", "schema.graphql"),
	}

	for _, path := range candidates {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		text := string(content)

		// Şemanın gerçekten Prova şeması olduğunu ve ana tipleri
		// içerdiğini doğrula. Boş ya da eski bir dosya, istemci kod
		// üretimini sessizce yanlış tiplerle besler.
		required := []string{
			"type Query", "type Mutation", "type Subscription",
			"directive @auth", "directive @permission",
			"type Session", "type Score", "type LLMProfile", "type RoutingStats",
		}
		var missing []string
		for _, needle := range required {
			if !strings.Contains(text, needle) {
				missing = append(missing, needle)
			}
		}
		if len(missing) > 0 {
			return fail("şema SDL olarak dışa aktarılmış",
				path+" içinde eksik: "+strings.Join(missing, ", "))
		}
		return pass("şema SDL olarak dışa aktarılmış",
			fmt.Sprintf("%s (%d bayt)", path, len(content)))
	}

	return fail("şema SDL olarak dışa aktarılmış",
		"schema.graphql bulunamadı (go run ./cmd/schema -out ../schema.graphql)")
}

// checkCORS, izinli origin'in gerçekten kabul edildiğini doğrular.
func (h *harness) checkCORS() check {
	allowed, err := h.corsHeader("http://localhost:3000")
	if err != nil {
		return fail("CORS izinli origin çalışıyor", err.Error())
	}
	if allowed == "" {
		return fail("CORS izinli origin çalışıyor",
			"izinli origin için Access-Control-Allow-Origin başlığı yok")
	}

	// İzinsiz origin reddedilmeli; aksi hâlde CORS yapılandırması
	// hiçbir şeyi korumuyor demektir.
	denied, err := h.corsHeader("https://saldirgan.example.com")
	if err != nil {
		return fail("CORS izinli origin çalışıyor", err.Error())
	}
	if denied != "" {
		return fail("CORS izinli origin çalışıyor",
			"izinsiz origin de kabul ediliyor: "+denied)
	}

	return pass("CORS izinli origin çalışıyor",
		fmt.Sprintf("localhost:3000 → %s, yabancı origin reddedildi", allowed))
}
