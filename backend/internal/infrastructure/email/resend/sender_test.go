package resend

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/masterfabric-go/masterfabric/internal/domain/notification/model"
	"github.com/masterfabric-go/masterfabric/internal/shared/config"
)

func testConfig(baseURL string) config.EmailConfig {
	return config.EmailConfig{
		Provider:    "resend",
		FromAddress: "noreply@mail.example.com",
		FromName:    "Prova",
		Timeout:     2 * time.Second,
		Resend: config.ResendConfig{
			APIKey:  "re_test_key",
			BaseURL: baseURL,
			Region:  "eu-west-1",
		},
	}
}

func testMessage() model.Message {
	return model.Message{
		To:       model.Address{Email: "user@corp.com"},
		Subject:  "Prova giriş doğrulama kodunuz",
		TextBody: "482913",
		HTMLBody: "<p>482913</p>",
		Tags:     map[string]string{"category": "login_code"},
	}
}

// A half-configured mailer surfaces as "nobody can log in", which is much
// harder to diagnose at runtime than at boot.
func TestNew_RequiresCredentialsAndSender(t *testing.T) {
	cfg := testConfig("https://api.resend.com")

	noKey := cfg
	noKey.Resend.APIKey = ""
	_, err := New(noKey)
	assert.Error(t, err)

	noFrom := cfg
	noFrom.FromAddress = ""
	_, err = New(noFrom)
	assert.Error(t, err)
}

func TestSender_SendsExpectedRequest(t *testing.T) {
	var (
		gotPath   string
		gotAuth   string
		gotBody   map[string]any
		gotMethod string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_123"}`))
	}))
	defer srv.Close()

	sender, err := New(testConfig(srv.URL))
	require.NoError(t, err)

	id, err := sender.Send(context.Background(), testMessage())
	require.NoError(t, err)

	assert.Equal(t, "msg_123", id)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/emails", gotPath)
	assert.Equal(t, "Bearer re_test_key", gotAuth)
	assert.Equal(t, "Prova <noreply@mail.example.com>", gotBody["from"])
	assert.Equal(t, []any{"user@corp.com"}, gotBody["to"])
	assert.Equal(t, "482913", gotBody["text"])
	assert.Equal(t, "<p>482913</p>", gotBody["html"])
}

func TestSender_ReportsProviderRejection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"statusCode":422,"name":"validation_error","message":"domain is not verified"}`))
	}))
	defer srv.Close()

	sender, err := New(testConfig(srv.URL))
	require.NoError(t, err)

	_, err = sender.Send(context.Background(), testMessage())

	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrDeliveryFailed)
	assert.Contains(t, err.Error(), "domain is not verified")
}

// The provider accepted the message; only the identifier is unreadable. The
// mail is on its way, so this must not fail the login request.
func TestSender_ToleratesUnreadableSuccessBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	sender, err := New(testConfig(srv.URL))
	require.NoError(t, err)

	id, err := sender.Send(context.Background(), testMessage())

	assert.NoError(t, err)
	assert.Empty(t, id)
}

func TestSender_RejectsInvalidMessageBeforeCallingProvider(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer srv.Close()

	sender, err := New(testConfig(srv.URL))
	require.NoError(t, err)

	msg := testMessage()
	msg.To = model.Address{}
	_, err = sender.Send(context.Background(), msg)

	assert.ErrorIs(t, err, model.ErrNoRecipient)
	assert.False(t, called)
}

// Resend restricts tag names and values to ASCII word characters. Sending
// anything else would get the whole message rejected.
func TestToTags_DropsUnsafeTags(t *testing.T) {
	tags := toTags(map[string]string{
		"category": "login_code",
		"kurum":    "Şube İstanbul",
		"":         "empty-name",
	})

	require.Len(t, tags, 1)
	assert.Equal(t, "category", tags[0].Name)
	assert.Equal(t, "login_code", tags[0].Value)
}

func TestSender_Name(t *testing.T) {
	sender, err := New(testConfig("https://api.resend.com"))
	require.NoError(t, err)
	assert.Equal(t, "resend", sender.Name())
}
