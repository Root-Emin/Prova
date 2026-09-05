package usecase

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	appService "github.com/masterfabric-go/masterfabric/internal/application/prova/service"
	"github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
	provaRepo "github.com/masterfabric-go/masterfabric/internal/domain/prova/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

type fakeSessionRepository struct {
	mu           sync.Mutex
	session      *model.Session
	turns        []*model.Turn
	appendCalls  int
	failAppendAt int
	appendErr    error
	deleteCalls  int
	findCalls    int
}

func (f *fakeSessionRepository) Create(context.Context, provaRepo.Scope, *model.Session) error {
	return nil
}
func (f *fakeSessionRepository) GetByID(context.Context, provaRepo.Scope, uuid.UUID) (*model.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.session, nil
}
func (f *fakeSessionRepository) ListByEmployee(context.Context, provaRepo.Scope, uuid.UUID, provaRepo.ListOptions) ([]*model.Session, error) {
	return []*model.Session{f.session}, nil
}
func (f *fakeSessionRepository) AppendTurn(_ context.Context, _ provaRepo.Scope, turn *model.Turn) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.appendCalls++
	if f.failAppendAt == f.appendCalls {
		return f.appendErr
	}
	turn.ID = uuid.New()
	turn.Index = len(f.turns)
	f.turns = append(f.turns, turn)
	f.session.TurnCount++
	return nil
}
func (f *fakeSessionRepository) FindTurnByRequestID(_ context.Context, _ provaRepo.Scope, sessionID, requestID uuid.UUID, role model.TurnRole) (*model.Turn, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.findCalls++
	for _, turn := range f.turns {
		if turn.SessionID == sessionID && turn.RequestID != nil && *turn.RequestID == requestID && turn.Role == role {
			return turn, nil
		}
	}
	return nil, domainErr.New(domainErr.ErrNotFound, "turn not found", nil)
}
func (f *fakeSessionRepository) DeleteTurn(_ context.Context, _ provaRepo.Scope, sessionID, turnID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteCalls++
	for i, turn := range f.turns {
		if turn.SessionID == sessionID && turn.ID == turnID {
			f.turns = append(f.turns[:i], f.turns[i+1:]...)
			f.session.TurnCount--
			return nil
		}
	}
	return domainErr.New(domainErr.ErrNotFound, "turn not found", nil)
}
func (f *fakeSessionRepository) ListTurns(context.Context, provaRepo.Scope, uuid.UUID) ([]*model.Turn, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*model.Turn(nil), f.turns...), nil
}
func (f *fakeSessionRepository) UpdateStatus(context.Context, provaRepo.Scope, uuid.UUID, model.SessionStatus, *time.Time) error {
	return nil
}
func (f *fakeSessionRepository) AnonymizeByEmployee(context.Context, uuid.UUID, time.Time) (int, error) {
	return 0, nil
}

type fakeVersionedRepo[T any] struct {
	doc      *T
	envelope func(*T) *model.Document
	validate func(*T) error
}

func (f *fakeVersionedRepo[T]) Create(_ context.Context, _ provaRepo.Scope, doc *T) error {
	f.doc = doc
	return nil
}
func (f *fakeVersionedRepo[T]) UpdateDraft(context.Context, provaRepo.Scope, *T) error { return nil }
func (f *fakeVersionedRepo[T]) NewVersion(context.Context, provaRepo.Scope, uuid.UUID, uuid.UUID, func(*T)) (*T, error) {
	return f.doc, nil
}
func (f *fakeVersionedRepo[T]) GetLatest(context.Context, provaRepo.Scope, uuid.UUID) (*T, error) {
	return f.doc, nil
}
func (f *fakeVersionedRepo[T]) GetLatestPublished(context.Context, provaRepo.Scope, uuid.UUID) (*T, error) {
	return f.doc, nil
}
func (f *fakeVersionedRepo[T]) GetVersion(context.Context, provaRepo.Scope, uuid.UUID, int) (*T, error) {
	return f.doc, nil
}
func (f *fakeVersionedRepo[T]) GetByID(context.Context, provaRepo.Scope, uuid.UUID) (*T, error) {
	return f.doc, nil
}
func (f *fakeVersionedRepo[T]) List(context.Context, provaRepo.Scope, provaRepo.ListOptions) ([]*T, error) {
	return []*T{f.doc}, nil
}
func (f *fakeVersionedRepo[T]) Publish(context.Context, provaRepo.Scope, uuid.UUID, int) (*T, error) {
	return f.doc, nil
}
func (f *fakeVersionedRepo[T]) Archive(context.Context, provaRepo.Scope, uuid.UUID, int) (*T, error) {
	return f.doc, nil
}

type fakePersonaClient struct {
	mu       sync.Mutex
	requests []appService.PersonaRequest
	response appService.PersonaResponse
	err      error
	wait     bool
	started  chan struct{}
}

func (f *fakePersonaClient) Respond(ctx context.Context, request appService.PersonaRequest) (appService.PersonaResponse, error) {
	f.mu.Lock()
	f.requests = append(f.requests, request)
	wait := f.wait
	err := f.err
	response := f.response
	started := f.started
	f.mu.Unlock()
	if started != nil {
		close(started)
	}
	if wait {
		<-ctx.Done()
		return appService.PersonaResponse{}, ctx.Err()
	}
	return response, err
}

func newSessionFixture() (*SessionUseCase, *fakeSessionRepository, *fakePersonaClient, provaRepo.Scope, uuid.UUID, uuid.UUID) {
	orgID, employeeID, sessionID := uuid.New(), uuid.New(), uuid.New()
	clock := func() time.Time { return time.Now() }
	character := model.NewCharacter(orgID, employeeID, clock)
	character.Name = "Ayşe"
	character.Persona = "Gecikmiş teslimat nedeniyle endişeli müşteri"
	character.Difficulty = model.DifficultyMedium
	character.BehaviorRules = []string{"Çözüm önerisini dinle."}
	character.HiddenFacts = []string{"Sipariş numarası 42"}
	scenario := model.NewScenario(orgID, employeeID, clock)
	scenario.Title = "Teslimat"
	scenario.Context = "Sipariş gecikmiş durumda."
	scenario.Objective = "Sorunun çözülmesini istiyor."
	rubric := model.NewRubric(orgID, employeeID, clock)
	rubric.Name = "Temel"
	rubric.Criteria = []model.Criterion{{Key: "empathy", Title: "Empati", MaxPoints: 1, Weight: 1}}
	session := &model.Session{
		ID: sessionID, OrgID: orgID, EmployeeID: employeeID, Status: model.SessionStatusActive,
		Scenario: scenario.Ref(), Character: character.Ref(), Rubric: rubric.Ref(), MaxTurns: 20,
	}
	repo := &fakeSessionRepository{session: session}
	client := &fakePersonaClient{response: appService.PersonaResponse{Text: "Konuyu birlikte çözebiliriz.", Emotion: "cooperative", ConversationState: "active"}}
	uc := NewSessionUseCase(SessionDeps{
		Sessions:      repo,
		Scenarios:     &fakeVersionedRepo[model.Scenario]{doc: scenario},
		Characters:    &fakeVersionedRepo[model.Character]{doc: character},
		Rubrics:       &fakeVersionedRepo[model.Rubric]{doc: rubric},
		PersonaClient: client,
	})
	return uc, repo, client, provaRepo.NewScope(orgID), employeeID, sessionID
}

func TestSubmitTurnUsesServerContextAndPersistsEmployeePersonaPair(t *testing.T) {
	uc, repo, client, scope, employeeID, sessionID := newSessionFixture()
	oldRequestID := uuid.New()
	repo.turns = []*model.Turn{{SessionID: sessionID, Role: model.TurnRoleEmployee, Text: "Önceki mesaj"}, {SessionID: sessionID, Role: model.TurnRoleCharacter, Text: "Önceki yanıt", RequestID: &oldRequestID}}
	repo.session.TurnCount = len(repo.turns)
	requestID := uuid.New()

	turn, err := uc.SubmitTurn(context.Background(), scope, employeeID, sessionID, requestID, "Çözüm için yardımcı olacağım.")
	if err != nil {
		t.Fatalf("SubmitTurn() error = %v", err)
	}
	if turn.Text != client.response.Text || len(repo.turns) != 4 {
		t.Fatalf("expected persisted persona pair, turn=%+v turns=%d", turn, len(repo.turns))
	}
	if len(client.requests) != 1 {
		t.Fatalf("persona client calls = %d, want 1", len(client.requests))
	}
	request := client.requests[0]
	if request.Persona.Name != "Ayşe" || request.Scenario.Title != "Teslimat" || request.EmployeeMessage != "Çözüm için yardımcı olacağım." {
		t.Fatalf("server mapping incomplete: %+v", request)
	}
	if len(request.ConversationHistory) != 2 || request.ConversationHistory[0].Role != "employee" || request.ConversationHistory[1].Role != "persona" {
		t.Fatalf("history mapping incorrect: %+v", request.ConversationHistory)
	}
	if repo.turns[2].RequestID == nil || repo.turns[3].RequestID == nil || *repo.turns[2].RequestID != requestID || *repo.turns[3].RequestID != requestID {
		t.Fatal("both persisted turns must carry the request id")
	}
}

func TestSubmitTurnAuthorizationAndStatePreventInference(t *testing.T) {
	tests := []struct {
		name   string
		status model.SessionStatus
		owner  uuid.UUID
		want   error
	}{
		{name: "wrong owner", status: model.SessionStatusActive, owner: uuid.New(), want: domainErr.ErrForbidden},
		{name: "completed", status: model.SessionStatusCompleted, want: domainErr.ErrConflict},
		{name: "scoring", status: model.SessionStatusScoring, want: domainErr.ErrConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, _, client, scope, employeeID, sessionID := newSessionFixture()
			uc.SessionDeps.Sessions.(*fakeSessionRepository).session.Status = tt.status
			if tt.owner != uuid.Nil {
				employeeID = tt.owner
			}
			_, err := uc.SubmitTurn(context.Background(), scope, employeeID, sessionID, uuid.New(), "Merhaba")
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
			if len(client.requests) != 0 {
				t.Fatal("inference must not be called before authorization/state checks")
			}
		})
	}
}

func TestSubmitTurnProviderFailureLeavesConversationUnchanged(t *testing.T) {
	for _, providerErr := range []error{appService.ErrPersonaUnavailable, appService.ErrPersonaTimeout} {
		providerErr := providerErr
		t.Run(providerErr.Error(), func(t *testing.T) {
			uc, repo, client, scope, employeeID, sessionID := newSessionFixture()
			client.err = providerErr
			_, err := uc.SubmitTurn(context.Background(), scope, employeeID, sessionID, uuid.New(), "Merhaba")
			if !errors.Is(err, domainErr.ErrInternal) || len(repo.turns) != 0 || repo.session.TurnCount != 0 {
				t.Fatalf("provider failure changed conversation: err=%v turns=%d count=%d", err, len(repo.turns), repo.session.TurnCount)
			}
		})
	}
}

func TestSubmitTurnPersistenceFailureRollsBackEmployeeTurn(t *testing.T) {
	uc, repo, client, scope, employeeID, sessionID := newSessionFixture()
	repo.failAppendAt = 2
	repo.appendErr = errors.New("persona persistence failed")
	_, err := uc.SubmitTurn(context.Background(), scope, employeeID, sessionID, uuid.New(), "Merhaba")
	if err == nil || len(repo.turns) != 0 || repo.session.TurnCount != 0 || repo.deleteCalls != 1 {
		t.Fatalf("persistence failure not compensated: err=%v turns=%d count=%d deletes=%d", err, len(repo.turns), repo.session.TurnCount, repo.deleteCalls)
	}
	if len(client.requests) != 1 {
		t.Fatal("provider should be called exactly once")
	}
}

func TestSubmitTurnDuplicateRequestReturnsPersistedPersonaWithoutInference(t *testing.T) {
	uc, repo, client, scope, employeeID, sessionID := newSessionFixture()
	requestID := uuid.New()
	repo.session.TurnCount = 2
	repo.turns = []*model.Turn{
		{ID: uuid.New(), SessionID: sessionID, RequestID: &requestID, Role: model.TurnRoleEmployee, Text: "Aynı istek"},
		{ID: uuid.New(), SessionID: sessionID, RequestID: &requestID, Role: model.TurnRoleCharacter, Text: "Önceden üretildi"},
	}
	turn, err := uc.SubmitTurn(context.Background(), scope, employeeID, sessionID, requestID, "Aynı istek")
	if err != nil || turn.Text != "Önceden üretildi" || len(client.requests) != 0 || repo.appendCalls != 0 {
		t.Fatalf("duplicate request was not idempotent: turn=%+v err=%v calls=%d appends=%d", turn, err, len(client.requests), repo.appendCalls)
	}
}

func TestSubmitTurnRejectsRequestIDReuseWithDifferentMessage(t *testing.T) {
	uc, repo, _, scope, employeeID, sessionID := newSessionFixture()
	requestID := uuid.New()
	repo.session.TurnCount = 2
	repo.turns = []*model.Turn{{SessionID: sessionID, RequestID: &requestID, Role: model.TurnRoleEmployee, Text: "ilk"}}
	_, err := uc.SubmitTurn(context.Background(), scope, employeeID, sessionID, requestID, "başka")
	if !errors.Is(err, domainErr.ErrConflict) {
		t.Fatalf("error = %v, want conflict", err)
	}
}

func TestSubmitTurnPropagatesContextCancellation(t *testing.T) {
	uc, repo, client, scope, employeeID, sessionID := newSessionFixture()
	client.wait = true
	client.started = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := uc.SubmitTurn(ctx, scope, employeeID, sessionID, uuid.New(), "Merhaba")
		done <- err
	}()
	<-client.started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
	if len(repo.turns) != 0 {
		t.Fatal("cancelled inference must not persist turns")
	}
}

func TestBuildPersonaRequestTruncatesHistoryDeterministically(t *testing.T) {
	_, _, _, _, _, _ = newSessionFixture()
	character := &model.Character{Name: "A", Persona: "P", Difficulty: model.DifficultyLow}
	scenario := &model.Scenario{Title: "S", Context: "C"}
	history := make([]*model.Turn, 25)
	for i := range history {
		role := model.TurnRoleEmployee
		if i%2 == 1 {
			role = model.TurnRoleCharacter
		}
		history[i] = &model.Turn{Role: role, Text: fmt.Sprintf("turn-%02d", i)}
	}
	request := buildPersonaRequest(uuid.New(), uuid.New(), character, scenario, history, "current")
	if len(request.ConversationHistory) != maxPersonaHistoryTurns || request.ConversationHistory[0].Content != "turn-05" || request.ConversationHistory[19].Content != "turn-24" {
		t.Fatalf("history truncation incorrect: first=%v last=%v len=%d", request.ConversationHistory[0], request.ConversationHistory[19], len(request.ConversationHistory))
	}
}

var _ provaRepo.SessionRepository = (*fakeSessionRepository)(nil)
var _ provaRepo.CharacterRepository = (*fakeVersionedRepo[model.Character])(nil)
var _ provaRepo.ScenarioRepository = (*fakeVersionedRepo[model.Scenario])(nil)
var _ provaRepo.RubricRepository = (*fakeVersionedRepo[model.Rubric])(nil)
var _ appService.PersonaInferenceClient = (*fakePersonaClient)(nil)
