// Package resend adapts the official Resend Go SDK to the notification.Sender
// port. Provider details remain in infrastructure; application code only sees
// a provider-neutral message.
package resend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	resendSDK "github.com/resend/resend-go/v4"

	"github.com/masterfabric-go/masterfabric/internal/domain/notification/model"
	"github.com/masterfabric-go/masterfabric/internal/shared/config"
)

const defaultBaseURL = "https://api.resend.com/"

// Sender delivers messages through Resend.
type Sender struct {
	client  *resendSDK.Client
	from    model.Address
	replyTo string
	apiKey  string
}

// New creates one configured SDK client for application bootstrap. The API key
// is retained only by the SDK client and is never included in errors or logs.
func New(cfg config.EmailConfig) (*Sender, error) {
	if strings.TrimSpace(cfg.Resend.APIKey) == "" {
		return nil, fmt.Errorf("resend: RESEND_API_KEY is empty")
	}
	from, err := model.ParseAddress(cfg.FromAddress, cfg.FromName)
	if err != nil {
		return nil, fmt.Errorf("resend: RESEND_FROM_EMAIL is invalid")
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	client := resendSDK.NewCustomClient(&http.Client{
		Timeout:   timeout,
		Transport: tolerateEmptySuccessBody(http.DefaultTransport),
	}, cfg.Resend.APIKey)
	baseURL := strings.TrimSpace(cfg.Resend.BaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/") + "/")
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("resend: RESEND_BASE_URL is invalid")
	}
	client.BaseURL = parsed

	return &Sender{client: client, from: from, replyTo: cfg.ReplyTo, apiKey: cfg.Resend.APIKey}, nil
}

// Name implements service.Sender.
func (s *Sender) Name() string { return config.ProviderResend }

// Send implements service.Sender.
func (s *Sender) Send(ctx context.Context, msg model.Message) (string, error) {
	if err := msg.Validate(); err != nil {
		return "", err
	}

	params := &resendSDK.SendEmailRequest{
		From:    s.from.String(),
		To:      []string{msg.To.Email},
		Subject: msg.Subject,
		Text:    msg.TextBody,
		Html:    msg.HTMLBody,
		ReplyTo: s.replyTo,
		Tags:    toTags(msg.Tags),
	}
	response, err := s.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		// The SDK error can contain provider diagnostics, but never the request
		// body. Redact defensively even if a proxy/provider echoes an
		// Authorization value in its response diagnostics.
		detail := strings.ReplaceAll(err.Error(), s.apiKey, "[REDACTED]")
		return "", fmt.Errorf("%w: resend: %s", model.ErrDeliveryFailed, detail)
	}
	if response == nil {
		return "", nil
	}
	return response.Id, nil
}

func toTags(tags map[string]string) []resendSDK.Tag {
	if len(tags) == 0 {
		return nil
	}
	out := make([]resendSDK.Tag, 0, len(tags))
	for name, value := range tags {
		if isTagSafe(name) && isTagSafe(value) {
			out = append(out, resendSDK.Tag{Name: name, Value: value})
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isTagSafe(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

// tolerateEmptySuccessBody preserves the adapter's historical behavior for a
// provider that accepted a send but returned an empty/non-JSON body. The SDK
// otherwise reports a decode error after the provider has already accepted the
// message, which would make a user retry and receive duplicate codes.
type successBodyTransport struct{ base http.RoundTripper }

func tolerateEmptySuccessBody(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return successBodyTransport{base: base}
}

func (t successBodyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil || resp == nil || resp.StatusCode == http.StatusNoContent {
		return resp, err
	}
	raw, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(raw)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 && !json.Valid(trimmed) {
		raw = []byte(`{"id":""}`)
	} else if json.Valid(trimmed) {
		// The SDK only decodes structured errors when this header is present.
		// Supplying it for a JSON provider response preserves the useful error
		// message without exposing the request body.
		resp.Header.Set("Content-Type", "application/json")
	}
	resp.Body = io.NopCloser(bytes.NewReader(raw))
	resp.ContentLength = int64(len(raw))
	return resp, nil
}
