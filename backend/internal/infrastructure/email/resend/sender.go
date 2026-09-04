// Package resend adapts the Resend HTTP API to the notification.Sender port.
//
// It speaks the Resend REST API directly rather than through a vendor SDK. A
// single POST is the whole integration, and keeping it dependency-free means
// the provider decision stays reversible: replacing this package is the entire
// cost of leaving Resend.
package resend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/masterfabric-go/masterfabric/internal/domain/notification/model"
	"github.com/masterfabric-go/masterfabric/internal/shared/config"
)

const (
	defaultBaseURL = "https://api.resend.com"
	sendPath       = "/emails"
	maxErrorBody   = 4 << 10
)

// Sender delivers messages through Resend.
type Sender struct {
	apiKey  string
	baseURL string
	from    model.Address
	replyTo string
	client  *http.Client
}

// New creates a Resend sender. It fails fast when credentials or the sender
// address are missing: a half-configured mailer surfaces as users unable to log
// in at all, which is far harder to diagnose at runtime than at boot.
func New(cfg config.EmailConfig) (*Sender, error) {
	if strings.TrimSpace(cfg.Resend.APIKey) == "" {
		return nil, fmt.Errorf("resend: RESEND_API_KEY is empty")
	}
	if strings.TrimSpace(cfg.FromAddress) == "" {
		return nil, fmt.Errorf("resend: EMAIL_FROM_ADDRESS is empty")
	}

	baseURL := strings.TrimRight(cfg.Resend.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &Sender{
		apiKey:  cfg.Resend.APIKey,
		baseURL: baseURL,
		from:    model.Address{Email: cfg.FromAddress, Name: cfg.FromName},
		replyTo: cfg.ReplyTo,
		client:  &http.Client{Timeout: timeout},
	}, nil
}

// Name implements service.Sender.
func (s *Sender) Name() string { return "resend" }

type sendRequest struct {
	From    string    `json:"from"`
	To      []string  `json:"to"`
	Subject string    `json:"subject"`
	Text    string    `json:"text,omitempty"`
	HTML    string    `json:"html,omitempty"`
	ReplyTo string    `json:"reply_to,omitempty"`
	Tags    []sendTag `json:"tags,omitempty"`
}

type sendTag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type sendResponse struct {
	ID string `json:"id"`
}

type apiError struct {
	Name       string `json:"name"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
}

// Send implements service.Sender.
func (s *Sender) Send(ctx context.Context, msg model.Message) (string, error) {
	if err := msg.Validate(); err != nil {
		return "", err
	}

	payload := sendRequest{
		From:    s.from.String(),
		To:      []string{msg.To.Email},
		Subject: msg.Subject,
		Text:    msg.TextBody,
		HTML:    msg.HTMLBody,
		ReplyTo: s.replyTo,
		Tags:    toTags(msg.Tags),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("resend: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+sendPath, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("resend: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: resend: %v", model.ErrDeliveryFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("%w: resend: %s", model.ErrDeliveryFailed, describeError(resp))
	}

	var out sendResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		// Resend accepted the message; only the identifier is unreadable. The
		// mail is on its way, so this must not fail the login request.
		return "", nil
	}
	return out.ID, nil
}

// describeError extracts a message from an error response without ever
// including the request body, which carries the login code.
func describeError(resp *http.Response) string {
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	if err != nil || len(raw) == 0 {
		return fmt.Sprintf("http %d", resp.StatusCode)
	}
	var apiErr apiError
	if err := json.Unmarshal(raw, &apiErr); err == nil && apiErr.Message != "" {
		return fmt.Sprintf("http %d: %s", resp.StatusCode, apiErr.Message)
	}
	return fmt.Sprintf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
}

// toTags converts neutral tags to Resend's list form. Resend restricts tag
// names and values to ASCII letters, digits, underscores and dashes, so
// anything else is dropped rather than sent and rejected.
func toTags(tags map[string]string) []sendTag {
	if len(tags) == 0 {
		return nil
	}
	out := make([]sendTag, 0, len(tags))
	for name, value := range tags {
		if isTagSafe(name) && isTagSafe(value) {
			out = append(out, sendTag{Name: name, Value: value})
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
