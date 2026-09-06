// Command smoke, ürünü sıfırdan uçtan uca oynatır.
//
// Doğrulamadan farkı: doğrulama her gereksinimi ayrı ayrı kanıtlar, duman
// testi tek bir kullanıcının yolculuğunu baştan sona yürür. Bir gereksinim
// tek başına çalışırken zincirin tamamının çalışmaması mümkündür — adımlar
// birbirinin çıktısına bağlı, ve o bağların koptuğunu ancak bu test görür.
//
// Zincir: yönetici davet eder → Desktop kod ister ve doğrular → cihaz yöneticide
// görünür → oturum başlat → birkaç konuşma sırası gönder → oturumu bitir → skoru
// al → alıntıları doğrula → skoru ez → veriyi dışa aktar → hesabı sil → kalıcı
// silmeyi tetikle.
//
// Çalıştırma: scripts/smoke.sh
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// step, zincirin tek bir adımı.
type step struct {
	number int
	name   string
	detail string
}

func main() {
	var (
		graphQL = flag.String("graphql", envOr("SMOKE_GRAPHQL_URL", "http://localhost:8080/graphql"), "GraphQL ucu")
		mailpit = flag.String("mailpit", envOr("SMOKE_MAILPIT_URL", "http://localhost:8025"), "Mailpit web API'si")
		mockLLM = flag.String("mockllm", envOr("SMOKE_MOCKLLM_URL", "http://localhost:8099"), "sahte LLM kontrol ucu")
	)
	flag.Parse()

	fmt.Println("Prova duman testi — uçtan uca zincir")
	fmt.Println(strings.Repeat("=", 72))

	runner := &smokeRunner{
		client: newClient(*graphQL, *mailpit, *mockLLM),
		runID:  randomHex(4),
	}

	if err := runner.run(); err != nil {
		fmt.Printf("\n\033[1;31m✗ ZİNCİR KOPTU\033[0m: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n" + strings.Repeat("=", 72))
	fmt.Printf("\033[1;32m✓ DUMAN TESTİ GEÇTİ\033[0m — %d adım, %s\n",
		len(runner.steps), runner.elapsed.Round(time.Millisecond))
}

// smokeRunner, zinciri yürütür ve adımları raporlar.
type smokeRunner struct {
	*client
	runID   string
	steps   []step
	elapsed time.Duration
}

// record, tamamlanan adımı yazdırır.
func (r *smokeRunner) record(name, detail string) {
	r.steps = append(r.steps, step{number: len(r.steps) + 1, name: name, detail: detail})
	fmt.Printf("\033[1;32m✓\033[0m %2d. %-38s %s\n", len(r.steps), name, detail)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
