package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// client, doğrulama betiğinin konuştuğu servisler.
//
// Doğrulama, ürünü İSTEMCİNİN gördüğü yerden test eder: GraphQL uçları,
// Mailpit'in web API'si ve sahte sağlayıcının kontrol uçları. İç paketlere
// dokunmaz, çünkü iç paketleri çağıran bir doğrulama, uçtan uca çalıştığını
// değil derlendiğini kanıtlar.
type client struct {
	graphQL string
	mailpit string
	mockLLM string
	http    *http.Client
}

func newClient(graphQL, mailpit, mockLLM string) *client {
	return &client{
		graphQL: graphQL,
		mailpit: mailpit,
		mockLLM: mockLLM,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// gqlResponse, GraphQL yanıtının ortak zarfı.
type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message    string         `json:"message"`
		Path       []any          `json:"path"`
		Extensions map[string]any `json:"extensions"`
	} `json:"errors"`
	// status, HTTP durum kodu. Taşıma katmanının reddettiği istekler
	// (iptal edilmiş token gibi) GraphQL hatası olarak değil 401 olarak
	// döner, ve doğrulamanın ikisini ayırt edebilmesi gerekir.
	status int
}

func (r gqlResponse) hasError(substring string) bool {
	for _, e := range r.Errors {
		if strings.Contains(strings.ToLower(e.Message), strings.ToLower(substring)) {
			return true
		}
	}
	return false
}

func (r gqlResponse) errorCode() string {
	for _, e := range r.Errors {
		if code, ok := e.Extensions["code"].(string); ok {
			return code
		}
	}
	return ""
}

func (r gqlResponse) firstError() string {
	if len(r.Errors) == 0 {
		return ""
	}
	return r.Errors[0].Message
}

// decode, data alanını hedefe çözer.
func (r gqlResponse) decode(target any) error {
	if len(r.Data) == 0 || string(r.Data) == "null" {
		return fmt.Errorf("veri yok: %s", r.firstError())
	}
	return json.Unmarshal(r.Data, target)
}

// mustQuery, GraphQL isteği gönderir ve GraphQL hatalarını da hata sayar.
//
// query() yalnızca taşıma hatasını döndürüyor; GraphQL hatası yanıtın
// içinde geliyor ve dönen error nil kalıyor. Bir kontrolün onu incelemeyi
// unutması, gerçekleşmemiş bir işlemi başarılı saymasına yol açar — bu
// doğrulamanın ilk sürümünde tam olarak bu oldu ve uygulanmamış bir
// mutation PASS almış gibi göründü.
func (c *client) mustQuery(token, query string) (gqlResponse, error) {
	resp, err := c.query(token, query)
	if err != nil {
		return resp, err
	}
	if len(resp.Errors) > 0 {
		return resp, fmt.Errorf("%s", resp.firstError())
	}
	return resp, nil
}

// query, GraphQL isteği gönderir. GraphQL hataları yanıtın içindedir;
// çağıran onları incelemek zorundadır (bkz. mustQuery).
func (c *client) query(token, query string) (gqlResponse, error) {
	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return gqlResponse{}, err
	}

	req, err := http.NewRequest(http.MethodPost, c.graphQL, bytes.NewReader(body))
	if err != nil {
		return gqlResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return gqlResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return gqlResponse{}, err
	}

	var decoded gqlResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return gqlResponse{status: resp.StatusCode}, fmt.Errorf("yanıt çözümlenemedi (HTTP %d): %s", resp.StatusCode, truncate(raw))
	}
	decoded.status = resp.StatusCode
	return decoded, nil
}

// rawPost, GraphQL uçuna ham gövde gönderir (batch testi için).
func (c *client) rawPost(body string) (int, string, error) {
	resp, err := c.http.Post(c.graphQL, "application/json", strings.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw), nil
}

// restStatus, eski REST uçlarının durum kodunu döndürür.
func (c *client) restStatus(method, path string) (int, error) {
	base := strings.TrimSuffix(c.graphQL, "/graphql")
	req, err := http.NewRequest(method, base+path, strings.NewReader("{}"))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

// mailBody, adrese giden son iletinin düz metin gövdesini döndürür.
func (c *client) mailBody(email string) (string, error) {
	searchURL := fmt.Sprintf("%s/api/v1/search?query=%s&limit=1",
		c.mailpit, url.QueryEscape("to:"+email))

	var list struct {
		Messages []struct {
			ID string `json:"ID"`
		} `json:"messages"`
	}
	if err := c.getJSON(searchURL, &list); err != nil {
		return "", err
	}
	if len(list.Messages) == 0 {
		return "", fmt.Errorf("%s adresine ileti yok", email)
	}

	var message struct {
		Text string `json:"Text"`
		HTML string `json:"HTML"`
	}
	if err := c.getJSON(fmt.Sprintf("%s/api/v1/message/%s", c.mailpit, list.Messages[0].ID), &message); err != nil {
		return "", err
	}
	return message.Text, nil
}

// mailCount reads Mailpit's recipient index without downloading each message.
// It lets the smoke chain prove that an administrative invite is provisioning
// only; the first actual delivery must be initiated by Desktop login.
func (c *client) mailCount(email string) (int, error) {
	searchURL := fmt.Sprintf("%s/api/v1/search?query=%s&limit=50",
		c.mailpit, url.QueryEscape("to:"+email))
	var list struct {
		Messages []json.RawMessage `json:"messages"`
	}
	if err := c.getJSON(searchURL, &list); err != nil {
		return 0, err
	}
	return len(list.Messages), nil
}

var (
	codePattern = regexp.MustCompile(`(?m)^\s*(\d{6})\s*$`)
	linkPattern = regexp.MustCompile(`(https?://\S*?/auth/magic\?token=[A-Za-z0-9_\-]+)`)
)

// loginCode, iletideki altı haneli kodu ayıklar.
func loginCode(body string) (string, bool) {
	match := codePattern.FindStringSubmatch(body)
	if match == nil {
		return "", false
	}
	return match[1], true
}

// magicLink, iletideki tek kullanımlık bağlantıyı ayıklar.
func magicLink(body string) (string, bool) {
	match := linkPattern.FindStringSubmatch(body)
	if match == nil {
		return "", false
	}
	return match[1], true
}

// magicToken, bağlantıdaki token'ı ayıklar.
func magicToken(link string) (string, bool) {
	parsed, err := url.Parse(link)
	if err != nil {
		return "", false
	}
	token := parsed.Query().Get("token")
	return token, token != ""
}

// breakModel, sahte sağlayıcıda bir modeli bilerek devre dışı bırakır.
func (c *client) breakModel(model string) error {
	return c.control("/_control/fail?model=" + url.QueryEscape(model))
}

// healModel, modeli tekrar çalışır hâle getirir.
func (c *client) healModel(model string) error {
	return c.control("/_control/heal?model=" + url.QueryEscape(model))
}

// lastPrompt, sağlayıcının gördüğü son kullanıcı mesajını döndürür.
func (c *client) lastPrompt() (string, error) {
	var out struct {
		Prompt string `json:"prompt"`
	}
	if err := c.getJSON(c.mockLLM+"/_control/last-prompt", &out); err != nil {
		return "", err
	}
	return out.Prompt, nil
}

func (c *client) control(path string) error {
	resp, err := c.http.Post(c.mockLLM+path, "application/json", nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("kontrol ucu HTTP %d döndü", resp.StatusCode)
	}
	return nil
}

func (c *client) getJSON(target string, out any) error {
	resp, err := c.http.Get(target)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(raw))
	}
	return json.Unmarshal(raw, out)
}

func truncate(raw []byte) string {
	const max = 300
	if len(raw) > max {
		return string(raw[:max]) + "…"
	}
	return string(raw)
}

// corsHeader, ön kontrol (preflight) isteğine dönen izin başlığını okur.
func (c *client) corsHeader(origin string) (string, error) {
	req, err := http.NewRequest(http.MethodOptions, c.graphQL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "content-type,authorization")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	return resp.Header.Get("Access-Control-Allow-Origin"), nil
}
