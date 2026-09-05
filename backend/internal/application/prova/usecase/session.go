// Package usecase, Prova'nın uygulama akışlarını barındırır.
package usecase

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	appService "github.com/masterfabric-go/masterfabric/internal/application/prova/service"
	auditService "github.com/masterfabric-go/masterfabric/internal/domain/audit/service"
	iamRepo "github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	provaModel "github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
	provaRepo "github.com/masterfabric-go/masterfabric/internal/domain/prova/repository"
	provaService "github.com/masterfabric-go/masterfabric/internal/domain/prova/service"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// SessionDeps, oturum akışlarının bağımlılıkları.
type SessionDeps struct {
	// Devices, oturumu başlatan cihazın hâlâ yetkili olduğunu doğrulamak
	// için. Nil bırakılabilir; o zaman kontrol atlanır.
	Devices       iamRepo.DeviceRepository
	Sessions      provaRepo.SessionRepository
	Scores        provaRepo.ScoreRepository
	Scenarios     provaRepo.ScenarioRepository
	Characters    provaRepo.CharacterRepository
	Rubrics       provaRepo.RubricRepository
	PersonaClient appService.PersonaInferenceClient
	Gateway       *appService.Gateway
	Events        SessionPublisher
	Audit         auditService.Recorder
	Log           *slog.Logger
}

// SessionPublisher, oturum olaylarını abonelere yayar.
type SessionPublisher interface {
	Publish(sessionID uuid.UUID, event SessionEvent)
}

// SessionEventType, yayılan olayın türü.
type SessionEventType string

const (
	EventTranscriptChunk SessionEventType = "TRANSCRIPT_CHUNK"
	EventRubricSignal    SessionEventType = "RUBRIC_SIGNAL"
	EventCharacterReply  SessionEventType = "CHARACTER_REPLY"
	EventScoringProgress SessionEventType = "SCORING_PROGRESS"
	EventScoreReady      SessionEventType = "SCORE_READY"
)

// SessionEvent, subscription kanalından geçen olay.
type SessionEvent struct {
	SessionID  uuid.UUID
	Type       SessionEventType
	Text       string
	Turn       *provaModel.Turn
	Signals    []string
	Progress   int
	Score      *provaModel.Score
	OccurredAt time.Time
}

// SessionUseCase, oturumu başlatır, yürütür ve bitirir.
type SessionUseCase struct {
	SessionDeps
	now       func() time.Time
	turnMu    sync.Mutex
	turnLocks map[uuid.UUID]*sync.Mutex
}

// NewSessionUseCase wires the use case.
func NewSessionUseCase(deps SessionDeps) *SessionUseCase {
	if deps.Audit == nil {
		deps.Audit = auditService.NoopRecorder{}
	}
	return &SessionUseCase{SessionDeps: deps, now: time.Now, turnLocks: make(map[uuid.UUID]*sync.Mutex)}
}

// Start, yayınlanmış bir senaryodan yeni oturum açar.
//
// Senaryo, karakter ve rubrik sürümleri oturum belgesine yazılır. Bu, ürünün
// sertifikasyon iddiasının dayandığı tek mekanizma: içerik yarın değişse bile
// bu oturum kendi sürümüyle puanlanır.
func (uc *SessionUseCase) Start(ctx context.Context, scope provaRepo.Scope, employeeID uuid.UUID, deviceID *uuid.UUID, scenarioLineage uuid.UUID) (*provaModel.Session, error) {
	// İptal edilmiş cihazdan oturum başlatılamaz. Token denylist'i zaten
	// iptali yakalar, ama o kontrol taşıma katmanında ve Redis'e bağlı;
	// sertifika taşıyan bir sınavın cihaz kontrolü, önbelleğin çalışmasına
	// bağlı bırakılamayacak kadar önemli.
	if err := uc.ensureDeviceUsable(ctx, employeeID, deviceID); err != nil {
		return nil, err
	}

	scenario, err := uc.Scenarios.GetLatestPublished(ctx, scope, scenarioLineage)
	if err != nil {
		return nil, err
	}

	// Karakter ve rubrik, senaryonun işaret ettiği TAM sürümden okunuyor.
	// Soy üzerinden en son yayınlanmışı almak, senaryonun oynandığı bileşimi
	// sessizce değiştirirdi.
	if _, err := uc.Characters.GetByID(ctx, scope, scenario.CharacterRef.VersionID); err != nil {
		return nil, err
	}
	if _, err := uc.Rubrics.GetByID(ctx, scope, scenario.RubricRef.VersionID); err != nil {
		return nil, err
	}

	session := &provaModel.Session{
		EmployeeID: employeeID,
		DeviceID:   deviceID,
		Status:     provaModel.SessionStatusActive,
		Scenario:   scenario.Ref(),
		Character:  scenario.CharacterRef,
		Rubric:     scenario.RubricRef,
		MaxTurns:   scenario.EffectiveMaxTurns(),
		StartedAt:  uc.now().UTC(),
	}
	if err := uc.Sessions.Create(ctx, scope, session); err != nil {
		return nil, err
	}

	uc.Audit.Record(ctx, auditService.Entry{
		OrgID:        scope.OrgID(),
		UserID:       &employeeID,
		Action:       auditService.ActionSessionStarted,
		ResourceType: "session",
		ResourceID:   session.ID.String(),
		Metadata: map[string]any{
			"scenario_version":  scenario.Version,
			"character_version": scenario.CharacterRef.Version,
			"rubric_version":    scenario.RubricRef.Version,
		},
	})

	return session, nil
}

// ensureDeviceUsable, oturumu başlatan cihazın hâlâ yetkili olduğunu doğrular.
func (uc *SessionUseCase) ensureDeviceUsable(ctx context.Context, employeeID uuid.UUID, deviceID *uuid.UUID) error {
	if deviceID == nil || uc.Devices == nil {
		// Web panelinin donanım kimliği yok; giriş yapar ama sertifika
		// taşıyan bir sınav barındıramaz. Bu kontrol onu engellemez,
		// engelleyen şey oturumun kendi kayıt kuralları.
		return nil
	}

	device, err := uc.Devices.GetByID(ctx, employeeID, *deviceID)
	if err != nil {
		if errors.Is(err, domainErr.ErrNotFound) {
			return domainErr.New(domainErr.ErrForbidden, "oturum başlatan cihaz tanınmıyor", nil)
		}
		return err
	}
	if device.IsRevoked() {
		return domainErr.New(domainErr.ErrForbidden, "bu cihazın yetkisi iptal edilmiş", nil)
	}
	return nil
}

// SubmitTurn, çalışanın mesajını server-side session context ile persona
// inference'a gönderir ve başarılı employee/persona çiftini kalıcılaştırır.
func (uc *SessionUseCase) SubmitTurn(ctx context.Context, scope provaRepo.Scope, employeeID, sessionID, requestID uuid.UUID, text string) (*provaModel.Turn, error) {
	session, err := uc.loadOwnedSession(ctx, scope, employeeID, sessionID)
	if err != nil {
		return nil, err
	}
	if !session.IsActive() {
		return nil, provaModel.ErrSessionNotActive
	}

	unlock := uc.lockTurn(sessionID)
	defer unlock()
	session, err = uc.loadOwnedSession(ctx, scope, employeeID, sessionID)
	if err != nil {
		return nil, err
	}
	if !session.IsActive() {
		return nil, provaModel.ErrSessionNotActive
	}
	if strings.TrimSpace(text) == "" {
		return nil, domainErr.New(domainErr.ErrValidation, "çalışan mesajı boş olamaz", nil)
	}
	if requestID == uuid.Nil {
		requestID = uuid.New()
	}

	// Authorization and state checks precede this lookup so an idempotency key
	// cannot be used to probe another employee's conversation.
	if existing, lookupErr := uc.Sessions.FindTurnByRequestID(ctx, scope, sessionID, requestID, provaModel.TurnRoleEmployee); lookupErr == nil {
		if existing.Text != text {
			return nil, domainErr.New(domainErr.ErrConflict, "request_id başka bir mesaj için kullanılmış", nil)
		}
		personaTurn, personaErr := uc.Sessions.FindTurnByRequestID(ctx, scope, sessionID, requestID, provaModel.TurnRoleCharacter)
		if personaErr != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "idempotent konuşma sırası eksik", personaErr)
		}
		return personaTurn, nil
	} else if !errors.Is(lookupErr, domainErr.ErrNotFound) {
		return nil, lookupErr
	}
	if !session.HasTurnsLeft() {
		return nil, provaModel.ErrSessionTurnLimit
	}

	character, _, scenario, err := uc.loadFrozenContent(ctx, scope, session)
	if err != nil {
		return nil, err
	}
	history, err := uc.Sessions.ListTurns(ctx, scope, sessionID)
	if err != nil {
		return nil, err
	}
	if uc.PersonaClient == nil {
		return nil, domainErr.New(domainErr.ErrInternal, "persona inference client yapılandırılmamış", nil)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	response, err := uc.PersonaClient.Respond(ctx, buildPersonaRequest(requestID, sessionID, character, scenario, history, text))
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, mapPersonaError(err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	requestIDCopy := requestID
	employeeTurn := &provaModel.Turn{
		SessionID: sessionID, RequestID: &requestIDCopy,
		Role: provaModel.TurnRoleEmployee, Text: text, Signals: []string{},
	}
	personaTurn := &provaModel.Turn{
		SessionID: sessionID, RequestID: &requestIDCopy,
		Role: provaModel.TurnRoleCharacter, Text: response.Text, Signals: []string{},
	}
	if err := uc.Sessions.AppendTurn(ctx, scope, employeeTurn); err != nil {
		return nil, err
	}
	if err := uc.Sessions.AppendTurn(ctx, scope, personaTurn); err != nil {
		// Rollback must still run when the request context was canceled after
		// the first insert; otherwise a client timeout could leave a dangling
		// employee turn without its persona reply.
		rollbackCtx, cancelRollback := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		rollbackErr := uc.Sessions.DeleteTurn(rollbackCtx, scope, sessionID, employeeTurn.ID)
		cancelRollback()
		if rollbackErr != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "konuşma sıraları geri alınamadı", errors.Join(err, rollbackErr))
		}
		return nil, err
	}

	// Publish only after both records exist. A provider or persistence failure
	// therefore cannot expose a transcript event for a half-written turn.
	uc.publish(SessionEvent{SessionID: sessionID, Type: EventTranscriptChunk, Text: text, Turn: employeeTurn, OccurredAt: uc.now().UTC()})
	uc.publish(SessionEvent{SessionID: sessionID, Type: EventCharacterReply, Text: response.Text, Turn: personaTurn, Signals: []string{}, OccurredAt: uc.now().UTC()})
	return personaTurn, nil
}

func (uc *SessionUseCase) lockTurn(sessionID uuid.UUID) func() {
	uc.turnMu.Lock()
	if uc.turnLocks == nil {
		uc.turnLocks = make(map[uuid.UUID]*sync.Mutex)
	}
	lock, ok := uc.turnLocks[sessionID]
	if !ok {
		lock = &sync.Mutex{}
		uc.turnLocks[sessionID] = lock
	}
	uc.turnMu.Unlock()
	lock.Lock()
	return lock.Unlock
}

func buildPersonaRequest(requestID, sessionID uuid.UUID, character *provaModel.Character, scenario *provaModel.Scenario, history []*provaModel.Turn, employeeMessage string) appService.PersonaRequest {
	personaHistory := make([]appService.ConversationTurn, 0, minPersonaHistory(len(history)))
	start := 0
	if len(history) > maxPersonaHistoryTurns {
		start = len(history) - maxPersonaHistoryTurns
	}
	for _, turn := range history[start:] {
		if turn == nil {
			continue
		}
		role := ""
		switch turn.Role {
		case provaModel.TurnRoleEmployee:
			role = "employee"
		case provaModel.TurnRoleCharacter:
			role = "persona"
		default:
			continue
		}
		personaHistory = append(personaHistory, appService.ConversationTurn{Role: role, Content: turn.Text})
	}
	return appService.PersonaRequest{
		RequestID: requestID.String(), SessionID: sessionID.String(), ScenarioID: scenario.ID.String(),
		Persona: appService.PersonaInput{
			ID: character.ID.String(), Name: character.Name, Role: "roleplay character", Description: character.Persona,
			BehavioralTraits: []string{string(character.Difficulty)}, CurrentState: map[string]any{},
		},
		Scenario: appService.ScenarioInput{
			ID: scenario.ID.String(), Title: scenario.Title, Context: scenario.Context, PersonaGoal: scenario.Objective,
			AllowedKnowledge: append([]string(nil), character.HiddenFacts...), BehavioralRules: append([]string(nil), character.BehaviorRules...),
		},
		ConversationHistory: personaHistory, EmployeeMessage: employeeMessage,
	}
}

const maxPersonaHistoryTurns = 20

func minPersonaHistory(length int) int {
	if length < maxPersonaHistoryTurns {
		return length
	}
	return maxPersonaHistoryTurns
}

func mapPersonaError(err error) error {
	switch {
	case errors.Is(err, appService.ErrPersonaValidation):
		return domainErr.New(domainErr.ErrValidation, "persona isteği geçersiz", nil)
	case errors.Is(err, appService.ErrPersonaTimeout):
		return domainErr.New(domainErr.ErrInternal, "persona yanıtı zaman aşımına uğradı", nil)
	case errors.Is(err, appService.ErrPersonaUnavailable):
		return domainErr.New(domainErr.ErrInternal, "persona servisi kullanılamıyor", nil)
	case errors.Is(err, appService.ErrPersonaInvalidModel):
		return domainErr.New(domainErr.ErrInternal, "persona yanıtı geçersiz", nil)
	default:
		return domainErr.New(domainErr.ErrInternal, "persona yanıtı alınamadı", nil)
	}
}

// End, oturumu bitirir ve rubriğe göre puanlar.
func (uc *SessionUseCase) End(ctx context.Context, scope provaRepo.Scope, employeeID, sessionID uuid.UUID) (*provaModel.Session, error) {
	session, err := uc.loadOwnedSession(ctx, scope, employeeID, sessionID)
	if err != nil {
		return nil, err
	}
	// Serialize lifecycle transition with SubmitTurn. This prevents scoring
	// from changing the session state between the employee and persona writes.
	unlock := uc.lockTurn(sessionID)
	defer unlock()
	session, err = uc.loadOwnedSession(ctx, scope, employeeID, sessionID)
	if err != nil {
		return nil, err
	}
	if session.Status == provaModel.SessionStatusCompleted {
		return session, nil
	}
	if !session.IsActive() {
		return nil, provaModel.ErrSessionNotActive
	}

	if err := uc.Sessions.UpdateStatus(ctx, scope, sessionID, provaModel.SessionStatusScoring, nil); err != nil {
		return nil, err
	}
	uc.publish(SessionEvent{
		SessionID: sessionID, Type: EventScoringProgress,
		Progress: 10, OccurredAt: uc.now().UTC(),
	})

	score, err := uc.scoreSession(ctx, scope, session)
	if err != nil {
		// Puanlama başarısız olursa oturum aktif hâle geri döndürülüyor:
		// "scoring" durumunda kilitli kalan bir oturum ne oynanabilir ne
		// puanlanabilir olurdu, ve çalışan tüm görüşmesini kaybederdi.
		if resetErr := uc.Sessions.UpdateStatus(ctx, scope, sessionID, provaModel.SessionStatusActive, nil); resetErr != nil {
			uc.Log.ErrorContext(ctx, "puanlama sonrası oturum durumu geri alınamadı",
				"session_id", sessionID, "error", resetErr)
		}
		return nil, err
	}

	endedAt := uc.now().UTC()
	if err := uc.Sessions.UpdateStatus(ctx, scope, sessionID, provaModel.SessionStatusCompleted, &endedAt); err != nil {
		return nil, err
	}
	session.Status = provaModel.SessionStatusCompleted
	session.EndedAt = &endedAt

	uc.publish(SessionEvent{
		SessionID: sessionID, Type: EventScoreReady,
		Progress: 100, Score: score, OccurredAt: endedAt,
	})
	uc.Audit.Record(ctx, auditService.Entry{
		OrgID:        scope.OrgID(),
		UserID:       &employeeID,
		Action:       auditService.ActionSessionEnded,
		ResourceType: "session",
		ResourceID:   sessionID.String(),
		Metadata: map[string]any{
			"passed":            score.Passed,
			"total":             score.Total,
			"unverified_quotes": score.UnverifiedQuoteCount(),
			"failed_mandatory":  score.FailedMandatoryKeys,
		},
	})

	return session, nil
}

// scoreSession, transkripti güçlü kademeye gönderir ve puanı yazar.
func (uc *SessionUseCase) scoreSession(ctx context.Context, scope provaRepo.Scope, session *provaModel.Session) (*provaModel.Score, error) {
	rubric, err := uc.Rubrics.GetByID(ctx, scope, session.Rubric.VersionID)
	if err != nil {
		return nil, err
	}
	scenario, err := uc.Scenarios.GetByID(ctx, scope, session.Scenario.VersionID)
	if err != nil {
		return nil, err
	}
	turns, err := uc.Sessions.ListTurns(ctx, scope, session.ID)
	if err != nil {
		return nil, err
	}
	if len(turns) == 0 {
		return nil, domainErr.New(domainErr.ErrValidation,
			"boş oturum puanlanamaz; en az bir konuşma sırası gerekiyor", nil)
	}

	messages := provaService.BuildScoringMessages(provaService.ScoringInput{
		Scenario: scenario,
		Rubric:   rubric,
		Turns:    turns,
	})

	uc.publish(SessionEvent{
		SessionID: session.ID, Type: EventScoringProgress,
		Progress: 40, OccurredAt: uc.now().UTC(),
	})

	result, err := uc.Gateway.Invoke(ctx, appService.Call{
		Scope:     scope,
		SessionID: &session.ID,
		Purpose:   appService.PurposeScoring,
		Messages:  messages,
		JSONMode:  true,
	})
	if err != nil {
		return nil, err
	}

	criteria, err := provaService.ParseScoring(result.Content, rubric)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "puanlama yanıtı çözümlenemedi", err)
	}

	uc.publish(SessionEvent{
		SessionID: session.ID, Type: EventScoringProgress,
		Progress: 80, OccurredAt: uc.now().UTC(),
	})

	// Alıntı doğrulaması, puan yazılmadan önce. Doğrulanmamış alıntı puanı
	// düşürmüyor ama işaretleniyor: karar insanın, ama karar veren insan
	// alıntının uydurma olduğunu görebilmeli.
	verifier := provaService.NewQuoteVerifier(turns)
	criteria = verifier.VerifyAll(criteria)

	score := &provaModel.Score{
		SessionID: session.ID,
		Criteria:  criteria,
		Model:     result.Model,
		RubricRef: session.Rubric,
		CreatedAt: uc.now().UTC(),
	}
	score.Evaluate(rubric.PassThreshold)

	if unverified := score.UnverifiedQuoteCount(); unverified > 0 {
		uc.Log.WarnContext(ctx, "puanlamada doğrulanamayan alıntı var",
			"session_id", session.ID, "count", unverified, "model", result.Model)
	}

	if err := uc.Scores.Create(ctx, scope, score); err != nil {
		return nil, err
	}
	return score, nil
}

// loadOwnedSession, oturumu okur ve sahipliğini doğrular.
//
// Sahiplik kontrolü burada, resolver'da değil: transkript kişisel veri
// taşıyor, ve "başkasının oturumunu okuyamazsın" kuralı her giriş yolunda
// aynı olmalı.
func (uc *SessionUseCase) loadOwnedSession(ctx context.Context, scope provaRepo.Scope, employeeID, sessionID uuid.UUID) (*provaModel.Session, error) {
	session, err := uc.Sessions.GetByID(ctx, scope, sessionID)
	if err != nil {
		return nil, err
	}
	if session.EmployeeID != employeeID {
		return nil, provaModel.ErrSessionNotOwned
	}
	return session, nil
}

// loadFrozenContent, oturumun dondurduğu sürümleri okur.
func (uc *SessionUseCase) loadFrozenContent(ctx context.Context, scope provaRepo.Scope, session *provaModel.Session) (*provaModel.Character, *provaModel.Rubric, *provaModel.Scenario, error) {
	character, err := uc.Characters.GetByID(ctx, scope, session.Character.VersionID)
	if err != nil {
		return nil, nil, nil, err
	}
	rubric, err := uc.Rubrics.GetByID(ctx, scope, session.Rubric.VersionID)
	if err != nil {
		return nil, nil, nil, err
	}
	scenario, err := uc.Scenarios.GetByID(ctx, scope, session.Scenario.VersionID)
	if err != nil {
		return nil, nil, nil, err
	}
	return character, rubric, scenario, nil
}

func (uc *SessionUseCase) publish(event SessionEvent) {
	if uc.Events == nil {
		return
	}
	uc.Events.Publish(event.SessionID, event)
}
