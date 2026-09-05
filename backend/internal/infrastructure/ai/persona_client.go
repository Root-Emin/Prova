// Package ai contains infrastructure adapters for the internal AI service.
package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	provaService "github.com/masterfabric-go/masterfabric/internal/application/prova/service"
)

const maxResponseBytes = 256 * 1024

// Client is a context-aware HTTP client for the versioned persona contract.
// It intentionally does not expose provider/model details to application code.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewPersonaClient(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *Client) Respond(ctx context.Context, request provaService.PersonaRequest) (provaService.PersonaResponse, error) {
	if err := validateRequest(request); err != nil {
		return provaService.PersonaResponse{}, err
	}
	body, err := json.Marshal(request)
	if err != nil {
		return provaService.PersonaResponse{}, newPersonaError(provaService.ErrPersonaValidation, "could not serialize persona request", err)
	}
	if err := ctx.Err(); err != nil {
		return provaService.PersonaResponse{}, classifyContextError(err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/persona/respond", strings.NewReader(string(body)))
	if err != nil {
		return provaService.PersonaResponse{}, newPersonaError(provaService.ErrPersonaUnavailable, "could not create persona request", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return provaService.PersonaResponse{}, classifyTransportError(ctx, err)
	}
	defer response.Body.Close()

	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if readErr != nil {
		return provaService.PersonaResponse{}, newPersonaError(provaService.ErrPersonaUnavailable, "could not read persona response", readErr)
	}
	if len(responseBody) > maxResponseBytes {
		return provaService.PersonaResponse{}, newPersonaError(provaService.ErrPersonaInvalidModel, "persona response is too large", nil)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return provaService.PersonaResponse{}, classifyHTTPStatus(response.StatusCode, responseBody)
	}

	var result provaService.PersonaResponse
	decoder := json.NewDecoder(strings.NewReader(string(responseBody)))
	if err := decoder.Decode(&result); err != nil {
		return provaService.PersonaResponse{}, newPersonaError(provaService.ErrPersonaInvalidModel, "persona response is malformed", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return provaService.PersonaResponse{}, newPersonaError(provaService.ErrPersonaInvalidModel, "persona response contains extra data", err)
	}
	if err := validateResponse(result); err != nil {
		return provaService.PersonaResponse{}, err
	}
	return result, nil
}

type personaError struct {
	kind error
	msg  string
	err  error
}

func (e *personaError) Error() string {
	if e.err == nil {
		return e.msg
	}
	return fmt.Sprintf("%s: %v", e.msg, e.err)
}

func (e *personaError) Unwrap() []error {
	if e.err == nil {
		return []error{e.kind}
	}
	return []error{e.kind, e.err}
}

func newPersonaError(kind error, message string, cause error) error {
	return &personaError{kind: kind, msg: message, err: cause}
}

func classifyContextError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return newPersonaError(provaService.ErrPersonaTimeout, "persona request timed out", err)
	}
	return newPersonaError(provaService.ErrPersonaUnavailable, "persona request was cancelled", err)
}

func classifyTransportError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return newPersonaError(provaService.ErrPersonaTimeout, "persona request timed out", err)
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return newPersonaError(provaService.ErrPersonaUnavailable, "persona request was cancelled", err)
	}
	return newPersonaError(provaService.ErrPersonaUnavailable, "persona service is unavailable", err)
}

func classifyHTTPStatus(status int, body []byte) error {
	// Read only the stable machine code. The server's message/body is never
	// copied into an error or log, because it may contain model/user content.
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &envelope)
	switch status {
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return newPersonaError(provaService.ErrPersonaTimeout, "persona service timed out", nil)
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return newPersonaError(provaService.ErrPersonaValidation, "persona request was rejected", nil)
	case http.StatusBadGateway:
		if envelope.Error.Code == "invalid_model_response" {
			return newPersonaError(provaService.ErrPersonaInvalidModel, "persona model response was invalid", nil)
		}
		return newPersonaError(provaService.ErrPersonaUnavailable, "persona service is unavailable", nil)
	case http.StatusServiceUnavailable, http.StatusInternalServerError:
		return newPersonaError(provaService.ErrPersonaUnavailable, "persona service is unavailable", nil)
	default:
		return newPersonaError(provaService.ErrPersonaUnavailable, "persona service returned an unexpected status", nil)
	}
}

func validateRequest(request provaService.PersonaRequest) error {
	if strings.TrimSpace(request.Persona.Name) == "" || strings.TrimSpace(request.Persona.Role) == "" || strings.TrimSpace(request.Persona.Description) == "" {
		return newPersonaError(provaService.ErrPersonaValidation, "persona identity is incomplete", nil)
	}
	if strings.TrimSpace(request.Scenario.Title) == "" || strings.TrimSpace(request.Scenario.Context) == "" {
		return newPersonaError(provaService.ErrPersonaValidation, "scenario context is incomplete", nil)
	}
	if strings.TrimSpace(request.EmployeeMessage) == "" {
		return newPersonaError(provaService.ErrPersonaValidation, "employee message is empty", nil)
	}
	if len(request.ConversationHistory) > 20 {
		return newPersonaError(provaService.ErrPersonaValidation, "conversation history is too long", nil)
	}
	for _, turn := range request.ConversationHistory {
		if turn.Role != "employee" && turn.Role != "persona" {
			return newPersonaError(provaService.ErrPersonaValidation, "conversation history role is invalid", nil)
		}
		if strings.TrimSpace(turn.Content) == "" {
			return newPersonaError(provaService.ErrPersonaValidation, "conversation history content is empty", nil)
		}
	}
	return nil
}

func validateResponse(response provaService.PersonaResponse) error {
	if strings.TrimSpace(response.Text) == "" {
		return newPersonaError(provaService.ErrPersonaInvalidModel, "persona response text is empty", nil)
	}
	validEmotion := map[string]bool{"neutral": true, "calm": true, "concerned": true, "frustrated": true, "angry": true, "relieved": true, "cooperative": true}
	validState := map[string]bool{"active": true, "resolved": true, "needs_clarification": true, "escalating": true, "ended": true}
	if !validEmotion[response.Emotion] || !validState[response.ConversationState] {
		return newPersonaError(provaService.ErrPersonaInvalidModel, "persona response has invalid structured fields", nil)
	}
	return nil
}

var _ provaService.PersonaInferenceClient = (*Client)(nil)
