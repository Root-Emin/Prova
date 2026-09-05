package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
)

// SessionRepository, oynanan oturumlar.
type SessionRepository interface {
	Create(ctx context.Context, scope Scope, session *model.Session) error
	GetByID(ctx context.Context, scope Scope, id uuid.UUID) (*model.Session, error)
	ListByEmployee(ctx context.Context, scope Scope, employeeID uuid.UUID, opts ListOptions) ([]*model.Session, error)

	// AppendTurn, konuşma sırasını yazar ve oturumun sayacını artırır.
	//
	// İkisi tek çağrıda yapılıyor: sayaç ayrı bir çağrıda artsaydı, iki
	// eşzamanlı gönderim aynı index'i alabilir ve alıntı doğrulaması yanlış
	// sıraya bakardı.
	AppendTurn(ctx context.Context, scope Scope, turn *model.Turn) error
	FindTurnByRequestID(ctx context.Context, scope Scope, sessionID, requestID uuid.UUID, role model.TurnRole) (*model.Turn, error)
	DeleteTurn(ctx context.Context, scope Scope, sessionID, turnID uuid.UUID) error
	ListTurns(ctx context.Context, scope Scope, sessionID uuid.UUID) ([]*model.Turn, error)

	// UpdateStatus, oturumun evresini değiştirir.
	UpdateStatus(ctx context.Context, scope Scope, id uuid.UUID, status model.SessionStatus, endedAt *time.Time) error

	// AnonymizeByEmployee, kalıcı silmenin MongoDB ayağı.
	//
	// Oturumlar silinmez, kimliksizleştirilir: çalışan kimliği anonim bir
	// değere çekilir ve çalışanın söyledikleri temizlenir. Puanlar kurum
	// istatistiği için korunur — silinen bir çalışanın verisi, kurumun
	// eğitim etkinliği ölçümünü geriye dönük bozmamalı.
	AnonymizeByEmployee(ctx context.Context, employeeID uuid.UUID, at time.Time) (int, error)
}

// ScoreRepository, puanlanmış sonuçlar.
type ScoreRepository interface {
	Create(ctx context.Context, scope Scope, score *model.Score) error
	GetByID(ctx context.Context, scope Scope, id uuid.UUID) (*model.Score, error)
	GetBySession(ctx context.Context, scope Scope, sessionID uuid.UUID) (*model.Score, error)
	ListBySessions(ctx context.Context, scope Scope, sessionIDs []uuid.UUID) (map[uuid.UUID]*model.Score, error)

	// ApplyOverride, yöneticinin ezme kaydını yazar.
	ApplyOverride(ctx context.Context, scope Scope, scoreID uuid.UUID, total float64, passed bool, override model.ScoreOverride) (*model.Score, error)
}

// RoutingRepository, yönlendirme kararları.
type RoutingRepository interface {
	Record(ctx context.Context, scope Scope, record *model.RoutingRecord) error
	List(ctx context.Context, scope Scope, sessionID *uuid.UUID, limit int) ([]*model.RoutingRecord, error)

	// Stats, kademe başına toplu istatistik döndürür.
	//
	// Toplama veritabanında yapılıyor: kayıtları çekip Go'da toplamak, bir
	// yılın kayıtlarını belleğe almak demek olurdu.
	Stats(ctx context.Context, scope Scope, from, to *time.Time) ([]model.TierStats, int, error)
}
