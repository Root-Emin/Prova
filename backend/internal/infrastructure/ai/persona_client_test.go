package ai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	provaService "github.com/masterfabric-go/masterfabric/internal/application/prova/service"
)

func testPersonaRequest() provaService.PersonaRequest {
	return provaService.PersonaRequest{
		RequestID: "req-1",
		SessionID: "session-1",
		Persona: provaService.PersonaInput{
			Name:        "Ayşe Yılmaz",
			Role:        "Müşteri",
			Description: "Gecikmiş teslimat nedeniyle endişeli bir müşteri.",
		},
		Scenario: provaService.ScenarioInput{
			Title:   "Gecikmiş teslimat",
			Context: "Sipariş beklenenden geç ulaştı.",
		},
		ConversationHistory: []provaService.ConversationTurn{{Role: "persona", Content: "Siparişim hâlâ gelmedi."}},
		EmployeeMessage:     "Sorunu çözmek için kimlik doğrulama adımını başlatalım.",
	}
}

func TestPersonaClientRespondsSuccessfully(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/persona/respond" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"Bu bilgiyi doğrulayalım.","emotion":"concerned","conversation_state":"needs_clarification","metadata":{"provider":"mock"}}`))
	}))
	defer server.Close()

	client := NewPersonaClient(server.URL, time.Second)
	response, err := client.Respond(context.Background(), testPersonaRequest())
	if err != nil {
		t.Fatalf("Respond() error = %v", err)
	}
	if response.Text == "" || response.Emotion != "concerned" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestPersonaClientContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewPersonaClient(server.URL, time.Second).Respond(ctx, testPersonaRequest())
	if !errors.Is(err, provaService.ErrPersonaUnavailable) {
		t.Fatalf("expected unavailable cancellation error, got %v", err)
	}
}

func TestPersonaClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	_, err := NewPersonaClient(server.URL, 10*time.Millisecond).Respond(context.Background(), testPersonaRequest())
	if !errors.Is(err, provaService.ErrPersonaTimeout) {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestPersonaClientConnectionFailure(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()
	_, err := NewPersonaClient(url, time.Second).Respond(context.Background(), testPersonaRequest())
	if !errors.Is(err, provaService.ErrPersonaUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
}

func TestPersonaClientMapsNon2xxWithoutReadingSensitiveBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"message":"employee secret should not be exposed"}}`))
	}))
	defer server.Close()
	_, err := NewPersonaClient(server.URL, time.Second).Respond(context.Background(), testPersonaRequest())
	if !errors.Is(err, provaService.ErrPersonaValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if errors.Is(err, provaService.ErrPersonaInvalidModel) || contains(err.Error(), "employee secret") {
		t.Fatalf("response body leaked or mapped incorrectly: %v", err)
	}
}

func TestPersonaClientRejectsMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"text":"ok","emotion":"calm"}`))
	}))
	defer server.Close()
	_, err := NewPersonaClient(server.URL, time.Second).Respond(context.Background(), testPersonaRequest())
	if !errors.Is(err, provaService.ErrPersonaInvalidModel) {
		t.Fatalf("expected invalid model response, got %v", err)
	}
}

func TestPersonaClientMapsInvalidModelErrorCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_model_response","message":"raw model output must not escape"}}`))
	}))
	defer server.Close()
	_, err := NewPersonaClient(server.URL, time.Second).Respond(context.Background(), testPersonaRequest())
	if !errors.Is(err, provaService.ErrPersonaInvalidModel) {
		t.Fatalf("expected invalid model error, got %v", err)
	}
	if contains(err.Error(), "raw model output") {
		t.Fatalf("error body leaked: %v", err)
	}
}

func TestPersonaClientRejectsInvalidRequestBeforeNetwork(t *testing.T) {
	request := testPersonaRequest()
	request.EmployeeMessage = "   "
	_, err := NewPersonaClient("http://127.0.0.1:1", time.Second).Respond(context.Background(), request)
	if !errors.Is(err, provaService.ErrPersonaValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

// Run with a live AI-01 service to verify the actual Go -> Python -> provider
// path: AI_SERVICE_URL=http://127.0.0.1:8090 go test ./internal/infrastructure/ai -run PythonSmoke
func TestPersonaClientPythonSmoke(t *testing.T) {
	baseURL := os.Getenv("AI_SERVICE_URL")
	if baseURL == "" {
		t.Skip("AI_SERVICE_URL is not set")
	}
	response, err := NewPersonaClient(baseURL, 3*time.Second).Respond(context.Background(), testPersonaRequest())
	if err != nil {
		t.Fatalf("live AI-01 service request failed: %v", err)
	}
	if response.Text == "" || response.Metadata["provider"] == nil {
		t.Fatalf("live AI-01 response is incomplete: %+v", response)
	}
}

func contains(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
