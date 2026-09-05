package service

import (
	"context"
	"errors"
)

// PersonaInferenceClient is the application boundary for the local persona
// service. The application knows this contract, not the Python provider or
// the model serving implementation behind it.
type PersonaInferenceClient interface {
	Respond(context.Context, PersonaRequest) (PersonaResponse, error)
}

var (
	ErrPersonaValidation   = errors.New("persona inference validation error")
	ErrPersonaUnavailable  = errors.New("persona inference provider unavailable")
	ErrPersonaTimeout      = errors.New("persona inference provider timeout")
	ErrPersonaInvalidModel = errors.New("persona inference returned an invalid model response")
)

type PersonaRequest struct {
	RequestID           string             `json:"request_id,omitempty"`
	SessionID           string             `json:"session_id,omitempty"`
	ScenarioID          string             `json:"scenario_id,omitempty"`
	Persona             PersonaInput       `json:"persona"`
	Scenario            ScenarioInput      `json:"scenario"`
	ConversationHistory []ConversationTurn `json:"conversation_history,omitempty"`
	EmployeeMessage     string             `json:"employee_message"`
}

type PersonaInput struct {
	ID               string         `json:"id,omitempty"`
	Name             string         `json:"name"`
	Role             string         `json:"role"`
	Description      string         `json:"description"`
	BehavioralTraits []string       `json:"behavioral_traits,omitempty"`
	CurrentState     map[string]any `json:"current_state,omitempty"`
}

type ScenarioInput struct {
	ID               string   `json:"id,omitempty"`
	Title            string   `json:"title"`
	Context          string   `json:"context"`
	PersonaGoal      string   `json:"persona_goal,omitempty"`
	AllowedKnowledge []string `json:"allowed_knowledge,omitempty"`
	BehavioralRules  []string `json:"behavioral_rules,omitempty"`
}

type ConversationTurn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type PersonaResponse struct {
	Text              string         `json:"text"`
	Emotion           string         `json:"emotion"`
	ConversationState string         `json:"conversation_state"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}
