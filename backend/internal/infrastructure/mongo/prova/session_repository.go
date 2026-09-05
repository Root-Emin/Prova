package prova

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/prova/repository"
	provaMongo "github.com/masterfabric-go/masterfabric/internal/infrastructure/mongo"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// SessionRepo, oturumlar ve konuşma sıraları.
type SessionRepo struct {
	sessions *mongo.Collection
	turns    *mongo.Collection
	now      func() time.Time
}

// NewSessionRepo wires the repository.
func NewSessionRepo(db *mongo.Database) *SessionRepo {
	return &SessionRepo{
		sessions: db.Collection(provaMongo.CollectionSessions),
		turns:    db.Collection(provaMongo.CollectionTurns),
		now:      time.Now,
	}
}

func scopedFilter(scope repository.Scope, conditions bson.M) (bson.M, error) {
	if scope.IsZero() {
		return nil, model.ErrOrgRequired
	}
	out := bson.M{}
	for k, v := range conditions {
		out[k] = v
	}
	out["org_id"] = scope.OrgID()
	return out, nil
}

// Create writes a new session.
func (r *SessionRepo) Create(ctx context.Context, scope repository.Scope, session *model.Session) error {
	if scope.IsZero() {
		return model.ErrOrgRequired
	}
	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}
	session.OrgID = scope.OrgID()
	if session.StartedAt.IsZero() {
		session.StartedAt = r.now().UTC()
	}
	if session.Status == "" {
		session.Status = model.SessionStatusActive
	}
	if _, err := r.sessions.InsertOne(ctx, session); err != nil {
		return domainErr.New(domainErr.ErrInternal, "oturum oluşturulamadı", err)
	}
	return nil
}

// GetByID returns one session.
func (r *SessionRepo) GetByID(ctx context.Context, scope repository.Scope, id uuid.UUID) (*model.Session, error) {
	condition, err := scopedFilter(scope, bson.M{"_id": id})
	if err != nil {
		return nil, err
	}
	var session model.Session
	if err := r.sessions.FindOne(ctx, condition).Decode(&session); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domainErr.New(domainErr.ErrNotFound, "oturum bulunamadı", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "oturum okunamadı", err)
	}
	return &session, nil
}

// ListByEmployee returns an employee's sessions, newest first.
func (r *SessionRepo) ListByEmployee(ctx context.Context, scope repository.Scope, employeeID uuid.UUID, opts repository.ListOptions) ([]*model.Session, error) {
	condition, err := scopedFilter(scope, bson.M{"employee_id": employeeID})
	if err != nil {
		return nil, err
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	cursor, err := r.sessions.Find(ctx, condition,
		options.Find().
			SetSort(bson.D{{Key: "started_at", Value: -1}}).
			SetSkip(int64(max(opts.Offset, 0))).
			SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "oturumlar listelenemedi", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var sessions []*model.Session
	for cursor.Next(ctx) {
		var s model.Session
		if err := cursor.Decode(&s); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "oturum çözümlenemedi", err)
		}
		sessions = append(sessions, &s)
	}
	if err := cursor.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "oturumlar listelenemedi", err)
	}
	return sessions, nil
}

// AppendTurn writes a turn and advances the session counter.
//
// Sıra numarası oturumun sayacından, tek bir atomik artırmayla alınıyor. İki
// eşzamanlı gönderim aynı numarayı alamaz; alsaydı, (org, session, index)
// benzersizlik index'i ikincisini reddederdi ama önce sayaç zaten bozulmuş
// olurdu.
func (r *SessionRepo) AppendTurn(ctx context.Context, scope repository.Scope, turn *model.Turn) error {
	condition, err := scopedFilter(scope, bson.M{
		"_id":    turn.SessionID,
		"status": model.SessionStatusActive,
	})
	if err != nil {
		return err
	}

	var session model.Session
	err = r.sessions.FindOneAndUpdate(ctx, condition,
		bson.M{"$inc": bson.M{"turn_count": 1}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&session)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return r.explainAppendFailure(ctx, scope, turn.SessionID)
		}
		return domainErr.New(domainErr.ErrInternal, "konuşma sırası eklenemedi", err)
	}

	if turn.ID == uuid.Nil {
		turn.ID = uuid.New()
	}
	turn.OrgID = scope.OrgID()
	turn.Index = session.TurnCount - 1
	if turn.CreatedAt.IsZero() {
		turn.CreatedAt = r.now().UTC()
	}
	if turn.Signals == nil {
		turn.Signals = []string{}
	}

	if _, err := r.turns.InsertOne(ctx, turn); err != nil {
		// Sayaç artmış ama sıra yazılamamışsa geri alınır. İşlem kullanmak
		// yerine telafi tercih edildi: tek belgelik bir artırma için çok
		// belgeli işlem, replica set zorunluluğu getirirdi.
		if _, undoErr := r.sessions.UpdateOne(ctx, bson.M{"_id": turn.SessionID},
			bson.M{"$inc": bson.M{"turn_count": -1}}); undoErr != nil {
			return domainErr.New(domainErr.ErrInternal,
				"konuşma sırası yazılamadı ve sayaç geri alınamadı", errors.Join(err, undoErr))
		}
		return domainErr.New(domainErr.ErrInternal, "konuşma sırası yazılamadı", err)
	}
	return nil
}

// FindTurnByRequestID returns one turn belonging to an idempotent request.
func (r *SessionRepo) FindTurnByRequestID(ctx context.Context, scope repository.Scope, sessionID, requestID uuid.UUID, role model.TurnRole) (*model.Turn, error) {
	condition, err := scopedFilter(scope, bson.M{
		"session_id": sessionID,
		"request_id": requestID,
		"role":       role,
	})
	if err != nil {
		return nil, err
	}
	var turn model.Turn
	if err := r.turns.FindOne(ctx, condition).Decode(&turn); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domainErr.New(domainErr.ErrNotFound, "konuşma sırası bulunamadı", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "konuşma sırası okunamadı", err)
	}
	return &turn, nil
}

// DeleteTurn removes a just-created turn and restores the session counter.
// It is the compensating half of the employee/persona pair when the second
// persistence operation fails on a Mongo deployment without transactions.
func (r *SessionRepo) DeleteTurn(ctx context.Context, scope repository.Scope, sessionID, turnID uuid.UUID) error {
	condition, err := scopedFilter(scope, bson.M{"_id": turnID, "session_id": sessionID})
	if err != nil {
		return err
	}
	result, err := r.turns.DeleteOne(ctx, condition)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "konuşma sırası geri alınamadı", err)
	}
	if result.DeletedCount == 0 {
		return domainErr.New(domainErr.ErrNotFound, "geri alınacak konuşma sırası bulunamadı", nil)
	}
	if _, err := r.sessions.UpdateOne(ctx, bson.M{
		"_id":        sessionID,
		"org_id":     scope.OrgID(),
		"status":     model.SessionStatusActive,
		"turn_count": bson.M{"$gt": 0},
	}, bson.M{"$inc": bson.M{"turn_count": -1}}); err != nil {
		return domainErr.New(domainErr.ErrInternal, "oturum sırası sayacı geri alınamadı", err)
	}
	return nil
}

func (r *SessionRepo) explainAppendFailure(ctx context.Context, scope repository.Scope, sessionID uuid.UUID) error {
	session, err := r.GetByID(ctx, scope, sessionID)
	if err != nil {
		return err
	}
	if !session.IsActive() {
		return model.ErrSessionNotActive
	}
	return model.ErrSessionTurnLimit
}

// ListTurns returns the transcript in order.
func (r *SessionRepo) ListTurns(ctx context.Context, scope repository.Scope, sessionID uuid.UUID) ([]*model.Turn, error) {
	condition, err := scopedFilter(scope, bson.M{"session_id": sessionID})
	if err != nil {
		return nil, err
	}
	cursor, err := r.turns.Find(ctx, condition, options.Find().SetSort(bson.D{{Key: "index", Value: 1}}))
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "konuşma sıraları listelenemedi", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var turns []*model.Turn
	for cursor.Next(ctx) {
		var t model.Turn
		if err := cursor.Decode(&t); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "konuşma sırası çözümlenemedi", err)
		}
		turns = append(turns, &t)
	}
	if err := cursor.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "konuşma sıraları listelenemedi", err)
	}
	return turns, nil
}

// UpdateStatus moves the session to a new phase.
func (r *SessionRepo) UpdateStatus(ctx context.Context, scope repository.Scope, id uuid.UUID, status model.SessionStatus, endedAt *time.Time) error {
	condition, err := scopedFilter(scope, bson.M{"_id": id})
	if err != nil {
		return err
	}
	set := bson.M{"status": status}
	if endedAt != nil {
		set["ended_at"] = endedAt.UTC()
	}
	result, err := r.sessions.UpdateOne(ctx, condition, bson.M{"$set": set})
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "oturum durumu güncellenemedi", err)
	}
	if result.MatchedCount == 0 {
		return domainErr.New(domainErr.ErrNotFound, "oturum bulunamadı", nil)
	}
	return nil
}

// AnonymizeByEmployee, kalıcı silmenin MongoDB ayağı.
//
// Kiracı sınırı olmadan çalışan tek sorgu budur ve bilinçlidir: silme talebi
// kullanıcıya aittir, kullanıcı ise birden fazla organizasyonda oturum
// oynamış olabilir. Sınırı burada uygulamak, bazı oturumların
// kimliksizleştirilmeden kalması demek olurdu.
func (r *SessionRepo) AnonymizeByEmployee(ctx context.Context, employeeID uuid.UUID, at time.Time) (int, error) {
	if employeeID == uuid.Nil {
		return 0, domainErr.New(domainErr.ErrValidation, "çalışan kimliği zorunlu", nil)
	}

	sessionFilter := bson.M{"employee_id": employeeID, "anonymized": bson.M{"$ne": true}}

	// Kimliksizleştirilecek oturumların kimlikleri önce toplanıyor: sıraların
	// temizlenmesi oturum kimliğine göre yapılıyor ve güncelleme sonrası
	// employee_id ile bulunamazlar.
	cursor, err := r.sessions.Find(ctx, sessionFilter, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return 0, domainErr.New(domainErr.ErrInternal, "kimliksizleştirilecek oturumlar okunamadı", err)
	}
	var ids []uuid.UUID
	for cursor.Next(ctx) {
		var row struct {
			ID uuid.UUID `bson:"_id"`
		}
		if err := cursor.Decode(&row); err != nil {
			_ = cursor.Close(ctx)
			return 0, domainErr.New(domainErr.ErrInternal, "oturum kimliği çözümlenemedi", err)
		}
		ids = append(ids, row.ID)
	}
	_ = cursor.Close(ctx)
	if len(ids) == 0 {
		return 0, nil
	}

	// Çalışanın söyledikleri temizleniyor; karakterin replikleri kalıyor.
	// Karakter metni kişisel veri taşımaz ve oturumun neye karşı oynandığını
	// gösteren tek kayıttır.
	if _, err := r.turns.UpdateMany(ctx,
		bson.M{"session_id": bson.M{"$in": ids}, "role": model.TurnRoleEmployee},
		bson.M{"$set": bson.M{"text": model.AnonymizedTurnText}},
	); err != nil {
		return 0, domainErr.New(domainErr.ErrInternal, "konuşma sıraları temizlenemedi", err)
	}

	result, err := r.sessions.UpdateMany(ctx, sessionFilter, bson.M{"$set": bson.M{
		"employee_id":   anonymousEmployeeID,
		"anonymized":    true,
		"anonymized_at": at.UTC(),
		"device_id":     nil,
	}})
	if err != nil {
		return 0, domainErr.New(domainErr.ErrInternal, "oturumlar kimliksizleştirilemedi", err)
	}
	return int(result.ModifiedCount), nil
}

// anonymousEmployeeID, kimliksizleştirilmiş oturumların çalışan kimliği.
//
// Sıfır UUID kullanılıyor: alanı boşaltmak yerine sabit bir değer yazmak,
// "kimliği silinmiş" ile "kimliği hiç olmayan" arasındaki farkı korur ve
// index'i geçerli tutar.
var anonymousEmployeeID = uuid.Nil

var _ repository.SessionRepository = (*SessionRepo)(nil)
