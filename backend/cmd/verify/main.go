// Command verify, on iki gereksinimi mekanik olarak kanıtlar.
//
// Her gereksinim tek tek PASS/FAIL basar; herhangi biri FAIL ise çıkış kodu
// 1'dir. Kontroller ürünü istemcinin gördüğü yerden yapar: GraphQL uçları,
// Mailpit'in web API'si, sahte sağlayıcının kontrol uçları ve iki
// veritabanı. İç paketleri çağıran bir doğrulama, uçtan uca çalıştığını
// değil yalnızca derlendiğini kanıtlar.
//
// Çalıştırma: scripts/verify.sh
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// requirement, doğrulanacak tek bir gereksinim.
type requirement struct {
	number int
	name   string
	run    func(*harness) []check
}

// check, bir gereksinimin tek bir kanıt maddesi.
type check struct {
	label  string
	passed bool
	detail string
}

func pass(label, detail string) check { return check{label: label, passed: true, detail: detail} }
func fail(label, detail string) check { return check{label: label, passed: false, detail: detail} }

// checkErr, hata varsa FAIL üretir.
func checkErr(label string, err error) check {
	if err != nil {
		return fail(label, err.Error())
	}
	return pass(label, "")
}

func main() {
	var (
		graphQL = flag.String("graphql", envOr("VERIFY_GRAPHQL_URL", "http://localhost:8080/graphql"), "GraphQL ucu")
		mailpit = flag.String("mailpit", envOr("VERIFY_MAILPIT_URL", "http://localhost:8025"), "Mailpit web API'si")
		mockLLM = flag.String("mockllm", envOr("VERIFY_MOCKLLM_URL", "http://localhost:8099"), "sahte LLM kontrol ucu")
		only    = flag.Int("only", 0, "yalnızca bu numaralı gereksinimi çalıştır")
	)
	flag.Parse()

	h, err := newHarness(newClient(*graphQL, *mailpit, *mockLLM))
	if err != nil {
		fmt.Fprintf(os.Stderr, "hazırlık başarısız: %v\n", err)
		os.Exit(1)
	}

	requirements := []requirement{
		{1, "GraphQL", verifyGraphQL},
		{2, "Object database (sürümleme)", verifyObjectDB},
		{3, "Web arayüzü (şema + CORS)", verifyWeb},
		{4, "Electron (cihaz uçları)", verifyElectron},
		{5, "İki LLM kademesi", verifyTwoTiers},
		{6, "Otomatik dağılım", verifyRouting},
		{7, "Hesap yaşam döngüsü", verifyLifecycle},
		{8, "Güvenlik", verifySecurity},
		{9, "E-posta doğrulama", verifyEmail},
		{10, "Kayıtlı cihaz", verifyDevice},
		{11, "LLM manipülasyonu", verifyLLMAdmin},
		{12, "KVKK", verifyKVKK},
	}

	fmt.Println("Prova gereksinim doğrulaması")
	fmt.Println(strings.Repeat("=", 72))

	failed := 0
	for _, req := range requirements {
		if *only != 0 && req.number != *only {
			continue
		}

		start := time.Now()
		checks := safeRun(h, req)

		requirementFailed := false
		for _, c := range checks {
			if !c.passed {
				requirementFailed = true
			}
		}
		if requirementFailed {
			failed++
		}

		status := "PASS"
		if requirementFailed {
			status = "FAIL"
		}
		fmt.Printf("\n[%s] %2d. %s (%s)\n", status, req.number, req.name, time.Since(start).Round(time.Millisecond))
		for _, c := range checks {
			mark := "  ✓"
			if !c.passed {
				mark = "  ✗"
			}
			line := fmt.Sprintf("%s %s", mark, c.label)
			if c.detail != "" {
				line += ": " + c.detail
			}
			fmt.Println(line)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 72))
	total := len(requirements)
	if *only != 0 {
		total = 1
	}
	if failed > 0 {
		fmt.Printf("SONUÇ: %d/%d PASS — %d gereksinim karşılanmadı\n", total-failed, total, failed)
		os.Exit(1)
	}
	fmt.Printf("SONUÇ: %d/%d PASS\n", total, total)
}

// safeRun, bir gereksinimin panic'ini FAIL'e çevirir.
//
// Panic eden bir kontrol, tüm doğrulamayı durdurup geri kalan on bir
// gereksinim hakkında hiçbir şey söylememeliydi.
func safeRun(h *harness, req requirement) (checks []check) {
	defer func() {
		if recovered := recover(); recovered != nil {
			checks = append(checks, fail("kontrol çöktü", fmt.Sprint(recovered)))
		}
	}()
	return req.run(h)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
