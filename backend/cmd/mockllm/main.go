// Command mockllm, OpenAI-uyumlu sahte bir sağlayıcıdır.
//
// Doğrulama ve duman testleri için var. Gerçek bir API anahtarı olmadan
// uçtan uca akışı çalıştırmayı ve — asıl önemlisi — hataları kasıtlı olarak
// üretmeyi sağlar: failover kaydının gerçekten oluştuğunu, hızlı sağlayıcı
// bilerek bozulmadan kanıtlamanın başka yolu yok.
//
// Çalıştırma:
//
//	go run ./cmd/mockllm -addr :8099
//
// Kontrol uçları:
//
//	POST /_control/fail?model=gpt-4o-mini   → o model için çağrılar hata döndürür
//	POST /_control/heal?model=gpt-4o-mini   → hata modunu kapatır
//	GET  /_control/stats                    → model başına çağrı sayısı
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

// state, hangi modellerin bilerek bozulduğunu ve çağrı sayaçlarını tutar.
type state struct {
	mu     sync.Mutex
	broken map[string]bool
	calls  map[string]int
}

func main() {
	addr := flag.String("addr", ":8099", "dinlenecek adres")
	flag.Parse()

	s := &state{broken: map[string]bool{}, calls: map[string]int{}}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", s.completions)
	mux.HandleFunc("/v1/models", s.models)
	mux.HandleFunc("/_control/fail", s.setBroken(true))
	mux.HandleFunc("/_control/heal", s.setBroken(false))
	mux.HandleFunc("/_control/stats", s.stats)
	mux.HandleFunc("/_control/reset", s.reset)

	log.Printf("mock llm dinliyor: %s", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}

type chatRequest struct {
	Model          string `json:"model"`
	Stream         bool   `json:"stream"`
	ResponseFormat *struct {
		Type string `json:"type"`
	} `json:"response_format"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

func (s *state) completions(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"message":"bozuk istek"}}`, http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.calls[req.Model]++
	broken := s.broken[req.Model]
	s.mu.Unlock()

	if broken {
		// 503, istemcinin yeniden denenebilir bir hata olarak göreceği kod.
		// Devre kesicinin ve failover'ın tetiklendiği yol tam olarak burası.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"message":"model bilerek devre dışı","type":"server_error"}}`))
		return
	}

	content := s.buildContent(req)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"model": req.Model,
		"choices": []map[string]any{
			{"message": map[string]string{"role": "assistant", "content": content}},
		},
		"usage": map[string]int{
			"prompt_tokens":     promptSize(req),
			"completion_tokens": len(content) / 4,
		},
	})
}

// buildContent, isteğin biçimine göre inandırıcı bir yanıt üretir.
//
// Puanlama isteği rubrik anahtarlarını içeriyor; sahte yanıt o anahtarları
// kullanır ve alıntıları TRANSKRİPT bölümünden birebir kopyalar. Uydurma
// alıntı üretmek de mümkün olmalı — alıntı doğrulamasının çalıştığını
// göstermek için bir kriter bilerek uydurma alıntı alır.
func (s *state) buildContent(req chatRequest) string {
	system, user := splitMessages(req)

	if strings.Contains(system, "değerlendiricisisin") {
		return scoringResponse(user)
	}
	return characterResponse(user)
}

// scoringResponse, rubrikteki her kriter için puan üretir.
func scoringResponse(user string) string {
	keys := extractKeys(user)
	quotes := extractEmployeeQuotes(user)

	type criterion struct {
		Key       string  `json:"key"`
		Points    float64 `json:"points"`
		Rationale string  `json:"rationale"`
		Quote     string  `json:"quote"`
		TurnIndex *int    `json:"turn_index"`
	}

	out := struct {
		Criteria []criterion `json:"criteria"`
	}{}

	for i, key := range keys {
		c := criterion{Key: key, Points: 5, Rationale: "Sahte değerlendirici: tam puan."}

		// PII kriterinde, transkriptte "veremem"/"veremiyorum" geçmiyorsa
		// düşük puan verilir. Bu, zorunlu kriter kuralının uçtan uca
		// doğrulanabilmesi için gerekli.
		if strings.Contains(key, "pii") && !mentionsRefusal(quotes) {
			c.Points = 1
			c.Rationale = "Sahte değerlendirici: kişisel veri talebi reddedilmemiş."
		}

		if i < len(quotes) {
			c.Quote = quotes[i].text
			index := quotes[i].index
			c.TurnIndex = &index
		} else if len(quotes) > 0 {
			c.Quote = quotes[0].text
			index := quotes[0].index
			c.TurnIndex = &index
		}

		// Son kriter bilerek uydurma alıntı alır: alıntı doğrulamasının
		// gerçekten çalıştığı ancak böyle gösterilebilir.
		if i == len(keys)-1 && len(keys) > 1 {
			c.Quote = "Bu cümle transkriptte hiç geçmiyor, sahte değerlendirici uydurdu."
		}

		out.Criteria = append(out.Criteria, c)
	}

	encoded, _ := json.Marshal(out)
	return string(encoded)
}

// characterResponse, karakter repliği ve sinyaller üretir.
func characterResponse(user string) string {
	reply := "Bakın, ben eşiyim. Bu bilgiyi almam gerekiyor, ısrar ediyorum."
	if strings.Contains(strings.ToLower(user), "iade") {
		reply = "Ayakkabıyı iade etmek istiyorum, ne yapmam gerekiyor?"
	}

	encoded, _ := json.Marshal(map[string]any{
		"reply":   reply,
		"signals": []string{},
	})
	return string(encoded)
}

type quote struct {
	index int
	text  string
}

// extractEmployeeQuotes, transkriptten çalışanın cümlelerini birebir alır.
func extractEmployeeQuotes(user string) []quote {
	var quotes []quote
	for _, line := range strings.Split(user, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "[") {
			continue
		}
		closing := strings.Index(line, "]")
		if closing < 0 {
			continue
		}
		var index int
		if _, err := fmt.Sscanf(line[1:closing], "%d", &index); err != nil {
			continue
		}
		rest := strings.TrimSpace(line[closing+1:])
		text, found := strings.CutPrefix(rest, "EMPLOYEE:")
		if !found {
			continue
		}
		quotes = append(quotes, quote{index: index, text: strings.TrimSpace(text)})
	}
	return quotes
}

// extractKeys, rubrik bölümündeki kriter anahtarlarını okur.
func extractKeys(user string) []string {
	var keys []string
	for _, line := range strings.Split(user, "\n") {
		if key, found := strings.CutPrefix(strings.TrimSpace(line), "- anahtar:"); found {
			keys = append(keys, strings.TrimSpace(key))
		}
	}
	return keys
}

func mentionsRefusal(quotes []quote) bool {
	for _, q := range quotes {
		lower := strings.ToLower(q.text)
		if strings.Contains(lower, "veremem") || strings.Contains(lower, "veremiyorum") ||
			strings.Contains(lower, "paylaşamam") || strings.Contains(lower, "paylaşamıyorum") {
			return true
		}
	}
	return false
}

func splitMessages(req chatRequest) (system, user string) {
	var systemParts, userParts []string
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemParts = append(systemParts, m.Content)
		} else {
			userParts = append(userParts, m.Content)
		}
	}
	return strings.Join(systemParts, "\n"), strings.Join(userParts, "\n")
}

func promptSize(req chatRequest) int {
	var chars int
	for _, m := range req.Messages {
		chars += len(m.Content)
	}
	return chars/4 + 1
}

func (s *state) models(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"gpt-4o-mini"},{"id":"gpt-4o"}]}`))
}

func (s *state) setBroken(broken bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		model := r.URL.Query().Get("model")
		if model == "" {
			http.Error(w, "model parametresi gerekli", http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		s.broken[model] = broken
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *state) stats(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	snapshot := make(map[string]int, len(s.calls))
	for k, v := range s.calls {
		snapshot[k] = v
	}
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snapshot)
}

func (s *state) reset(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	s.broken = map[string]bool{}
	s.calls = map[string]int{}
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}
