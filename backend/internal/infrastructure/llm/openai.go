// Package llm, OpenAI-uyumlu sağlayıcılar için HTTP istemcisini barındırır.
//
// Tek bir istemci yeterli: OpenAI, Azure OpenAI, Together, Groq, vLLM ve
// Ollama'nın uyumluluk uçları aynı gövdeyi konuşuyor. Sağlayıcı değiştirmek
// base URL ve model adını değiştirmek demek, ve ikisi de LLM profilinden
// geliyor.
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/masterfabric-go/masterfabric/internal/domain/prova/service"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// maxErrorBody, sağlayıcı hatasından loga alınacak gövdenin üst sınırı.
// Tam gövdeyi almak, bir sağlayıcının hata mesajına koyduğu prompt'u loga
// yazmak demek olabilir.
const maxErrorBody = 2 << 10

// Client, OpenAI-uyumlu sohbet tamamlama istemcisi.
type Client struct {
	http *http.Client
}

// NewClient wires the client with a request timeout.
func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &Client{http: &http.Client{Timeout: timeout}}
}

// chatRequest, /chat/completions gövdesi.
type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	Temperature    float64         `json:"temperature"`
	TopP           float64         `json:"top_p"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	Stream         bool            `json:"stream,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	// StreamOptions, akışta token sayısının da dönmesini ister. Olmadan
	// akışlı çağrıların maliyeti hesaplanamaz ve dağılım istatistiği
	// yarım kalır.
	StreamOptions *streamOptions `json:"stream_options,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message chatMessage `json:"message"`
		Delta   chatMessage `json:"delta"`
	} `json:"choices"`
	Usage *usage `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

// Complete implements service.Provider.
func (c *Client) Complete(ctx context.Context, req service.CompletionRequest) (*service.CompletionResponse, error) {
	started := time.Now()

	body, err := c.post(ctx, req.Target, "/chat/completions", buildRequest(req, false))
	if err != nil {
		return nil, err
	}
	defer func() { _ = body.Close() }()

	var decoded chatResponse
	if err := json.NewDecoder(body).Decode(&decoded); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "sağlayıcı yanıtı çözümlenemedi", err)
	}
	if decoded.Error != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "sağlayıcı hatası: "+decoded.Error.Message, nil)
	}
	if len(decoded.Choices) == 0 {
		return nil, domainErr.New(domainErr.ErrInternal, "sağlayıcı boş yanıt döndürdü", nil)
	}

	return &service.CompletionResponse{
		Content: decoded.Choices[0].Message.Content,
		Usage:   mapUsage(decoded.Usage, req),
		Model:   orDefault(decoded.Model, req.Model),
		Latency: time.Since(started),
	}, nil
}

// Stream implements service.Provider.
//
// Server-sent events okunuyor. Akış, karakterin yanıtını parça parça
// göstermek için var: tam yanıtı bekleyen bir arayüz, rol yapma hissini
// bekleme ekranına çevirir.
func (c *Client) Stream(ctx context.Context, req service.CompletionRequest, onDelta func(string) error) (*service.CompletionResponse, error) {
	started := time.Now()

	body, err := c.post(ctx, req.Target, "/chat/completions", buildRequest(req, true))
	if err != nil {
		return nil, err
	}
	defer func() { _ = body.Close() }()

	var (
		content strings.Builder
		final   chatResponse
	)

	scanner := bufio.NewScanner(body)
	// Uzun bir SSE satırı varsayılan 64KB tamponu aşabilir; JSON modda tek
	// parçada gelen büyük bir yanıt tam olarak bunu yapar.
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		payload, found := strings.CutPrefix(line, "data:")
		if !found {
			continue
		}
		payload = strings.TrimSpace(payload)
		if payload == "" || payload == "[DONE]" {
			continue
		}

		var chunk chatResponse
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			// Tek bir bozuk parça akışı düşürmemeli; sağlayıcılar arada
			// yorum satırı ve keep-alive gönderiyor.
			continue
		}
		if chunk.Error != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "sağlayıcı hatası: "+chunk.Error.Message, nil)
		}
		if chunk.Usage != nil {
			final.Usage = chunk.Usage
		}
		if chunk.Model != "" {
			final.Model = chunk.Model
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		content.WriteString(delta)
		if onDelta != nil {
			if err := onDelta(delta); err != nil {
				return nil, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "sağlayıcı akışı kesildi", err)
	}

	return &service.CompletionResponse{
		Content: content.String(),
		Usage:   mapUsage(final.Usage, req),
		Model:   orDefault(final.Model, req.Model),
		Latency: time.Since(started),
	}, nil
}

// Health implements service.Provider.
//
// /models uçuna bakıyor: bir tamamlama isteği göndermek sağlığı doğrulardı ama
// her kontrolde para harcardı, ve devre kesici bunu düzenli olarak çağırıyor.
func (c *Client) Health(ctx context.Context, target service.Target) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint(target.BaseURL, "/models"), nil)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "sağlık isteği kurulamadı", err)
	}
	if target.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+target.APIKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "sağlayıcıya ulaşılamadı", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorBody))

	if resp.StatusCode >= http.StatusInternalServerError {
		return domainErr.New(domainErr.ErrInternal,
			fmt.Sprintf("sağlayıcı sağlıksız: HTTP %d", resp.StatusCode), nil)
	}
	return nil
}

func (c *Client) post(ctx context.Context, target service.Target, path string, payload any) (io.ReadCloser, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "istek gövdesi kurulamadı", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint(target.BaseURL, path), bytes.NewReader(encoded))
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "istek kurulamadı", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if target.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+target.APIKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "sağlayıcıya ulaşılamadı", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		_ = resp.Body.Close()
		return nil, domainErr.New(domainErr.ErrInternal,
			fmt.Sprintf("sağlayıcı HTTP %d döndü", resp.StatusCode),
			fmt.Errorf("%s", strings.TrimSpace(string(snippet))))
	}
	return resp.Body, nil
}

func buildRequest(req service.CompletionRequest, stream bool) chatRequest {
	messages := make([]chatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		messages = append(messages, chatMessage{Role: string(m.Role), Content: m.Content})
	}

	out := chatRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxTokens,
		Stream:      stream,
	}
	if req.JSONMode {
		out.ResponseFormat = &responseFormat{Type: "json_object"}
	}
	if stream {
		out.StreamOptions = &streamOptions{IncludeUsage: true}
	}
	return out
}

// mapUsage, sağlayıcının bildirdiği token sayılarını çevirir.
//
// Sağlayıcı token bildirmezse kaba bir tahmin üretilir. Sıfır bırakmak
// maliyeti sıfır gösterirdi, ve "hepsi güçlü kademeye gitseydi" karşılaştırması
// da sıfıra bölünürdü — dağılım iddiasının kanıtı sessizce yok olurdu.
func mapUsage(u *usage, req service.CompletionRequest) service.Usage {
	if u != nil && (u.PromptTokens > 0 || u.CompletionTokens > 0) {
		return service.Usage{InputTokens: u.PromptTokens, OutputTokens: u.CompletionTokens}
	}
	var promptChars int
	for _, m := range req.Messages {
		promptChars += len(m.Content)
	}
	return service.Usage{
		InputTokens:  estimateTokens(promptChars),
		OutputTokens: 0,
	}
}

// estimateTokens, karakter sayısından kaba token tahmini.
//
// Dört karakter ≈ bir token, İngilizce için yaygın bir yaklaşıklık. Türkçe'de
// gerçek oran daha düşük olduğu için bu tahmin maliyeti olduğundan az
// gösterir; bu yüzden yalnızca sağlayıcı hiç bildirmediğinde kullanılıyor ve
// kayıtta tahmin olduğu ayırt edilebilsin diye çıplak sıfır yerine değer
// üretiliyor.
func estimateTokens(chars int) int {
	if chars <= 0 {
		return 0
	}
	return chars/4 + 1
}

func endpoint(baseURL, path string) string {
	return strings.TrimRight(baseURL, "/") + path
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

var _ service.Provider = (*Client)(nil)
