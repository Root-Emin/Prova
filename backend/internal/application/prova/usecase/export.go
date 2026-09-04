package usecase

import (
	"context"

	"github.com/google/uuid"
	provaModel "github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
	provaRepo "github.com/masterfabric-go/masterfabric/internal/domain/prova/repository"
)

// SessionExporter, bir çalışanın oturumlarını, transkriptlerini ve puanlarını
// dışa aktarma belgesine çevirir.
//
// Kimlik domain'indeki hesap yaşam döngüsü buna bir port üzerinden bağlanıyor;
// doğrudan bağlanmak, kimlik katmanını eğitim içeriğine bağımlı kılardı.
type SessionExporter struct {
	sessions provaRepo.SessionRepository
	scores   provaRepo.ScoreRepository
	// orgs, çalışanın oturumlarının hangi kiracılarda aranacağını söyler.
	// Bir kullanıcı birden fazla organizasyonda oturum oynamış olabilir ve
	// dışa aktarma hepsini kapsamalı.
	orgs OrgLister
}

// OrgLister, kullanıcının üye olduğu organizasyonları döndürür.
type OrgLister interface {
	OrgsForUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

// NewSessionExporter wires the exporter.
func NewSessionExporter(sessions provaRepo.SessionRepository, scores provaRepo.ScoreRepository, orgs OrgLister) *SessionExporter {
	return &SessionExporter{sessions: sessions, scores: scores, orgs: orgs}
}

// maxExportSessions, dışa aktarmaya girecek oturum üst sınırı.
const maxExportSessions = 1000

// ExportForEmployee implements the account use case's ExportSource port.
func (e *SessionExporter) ExportForEmployee(ctx context.Context, employeeID uuid.UUID) (map[string]any, error) {
	orgIDs, err := e.orgs.OrgsForUser(ctx, employeeID)
	if err != nil {
		return nil, err
	}

	sessionRows := make([]map[string]any, 0)
	for _, orgID := range orgIDs {
		scope := provaRepo.NewScope(orgID)

		sessions, err := e.sessions.ListByEmployee(ctx, scope, employeeID, provaRepo.ListOptions{Limit: maxExportSessions})
		if err != nil {
			return nil, err
		}
		if len(sessions) == 0 {
			continue
		}

		ids := make([]uuid.UUID, 0, len(sessions))
		for _, s := range sessions {
			ids = append(ids, s.ID)
		}
		scores, err := e.scores.ListBySessions(ctx, scope, ids)
		if err != nil {
			return nil, err
		}

		for _, session := range sessions {
			turns, err := e.sessions.ListTurns(ctx, scope, session.ID)
			if err != nil {
				return nil, err
			}
			sessionRows = append(sessionRows, exportSession(session, turns, scores[session.ID]))
		}
	}

	return map[string]any{"sessions": sessionRows}, nil
}

func exportSession(session *provaModel.Session, turns []*provaModel.Turn, score *provaModel.Score) map[string]any {
	transcript := make([]map[string]any, 0, len(turns))
	for _, t := range turns {
		transcript = append(transcript, map[string]any{
			"index":      t.Index,
			"role":       t.Role,
			"text":       t.Text,
			"signals":    t.Signals,
			"created_at": t.CreatedAt,
		})
	}

	row := map[string]any{
		"id":                session.ID,
		"organization_id":   session.OrgID,
		"status":            session.Status,
		"scenario_version":  session.Scenario.Version,
		"character_version": session.Character.Version,
		"rubric_version":    session.Rubric.Version,
		"started_at":        session.StartedAt,
		"ended_at":          session.EndedAt,
		"transcript":        transcript,
	}

	if score != nil {
		criteria := make([]map[string]any, 0, len(score.Criteria))
		for _, c := range score.Criteria {
			criteria = append(criteria, map[string]any{
				"criterion_key":  c.CriterionKey,
				"title":          c.Title,
				"points":         c.Points,
				"max_points":     c.MaxPoints,
				"mandatory":      c.Mandatory,
				"rationale":      c.Rationale,
				"quote":          c.Quote,
				"quote_verified": c.QuoteVerified,
			})
		}
		row["score"] = map[string]any{
			"total":                 score.Total,
			"max_total":             score.MaxTotal,
			"passed":                score.Passed,
			"failed_mandatory_keys": score.FailedMandatoryKeys,
			"model":                 score.Model,
			"criteria":              criteria,
			"overridden":            score.Override != nil,
			"created_at":            score.CreatedAt,
		}
	}

	return row
}
