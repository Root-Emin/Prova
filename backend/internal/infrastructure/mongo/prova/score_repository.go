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

// ScoreRepo, puanlanmış sonuçlar.
type ScoreRepo struct {
	coll *mongo.Collection
	now  func() time.Time
}

// NewScoreRepo wires the repository.
func NewScoreRepo(db *mongo.Database) *ScoreRepo {
	return &ScoreRepo{coll: db.Collection(provaMongo.CollectionScores), now: time.Now}
}

// Create writes the score for a session.
func (r *ScoreRepo) Create(ctx context.Context, scope repository.Scope, score *model.Score) error {
	if scope.IsZero() {
		return model.ErrOrgRequired
	}
	if score.ID == uuid.Nil {
		score.ID = uuid.New()
	}
	score.OrgID = scope.OrgID()
	if score.CreatedAt.IsZero() {
		score.CreatedAt = r.now().UTC()
	}
	if score.FailedMandatoryKeys == nil {
		score.FailedMandatoryKeys = []string{}
	}

	if _, err := r.coll.InsertOne(ctx, score); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			// (org_id, session_id) benzersiz: iki eşzamanlı bitirme isteği
			// aynı oturuma iki farklı puan yazamaz.
			return model.ErrSessionAlreadyScored
		}
		return domainErr.New(domainErr.ErrInternal, "puan yazılamadı", err)
	}
	return nil
}

// GetByID returns one score.
func (r *ScoreRepo) GetByID(ctx context.Context, scope repository.Scope, id uuid.UUID) (*model.Score, error) {
	return r.findOne(ctx, scope, bson.M{"_id": id})
}

// GetBySession returns the score attached to a session.
func (r *ScoreRepo) GetBySession(ctx context.Context, scope repository.Scope, sessionID uuid.UUID) (*model.Score, error) {
	return r.findOne(ctx, scope, bson.M{"session_id": sessionID})
}

func (r *ScoreRepo) findOne(ctx context.Context, scope repository.Scope, conditions bson.M) (*model.Score, error) {
	condition, err := scopedFilter(scope, conditions)
	if err != nil {
		return nil, err
	}
	var score model.Score
	if err := r.coll.FindOne(ctx, condition).Decode(&score); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domainErr.New(domainErr.ErrNotFound, "puan bulunamadı", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "puan okunamadı", err)
	}
	return &score, nil
}

// ListBySessions returns scores keyed by session.
//
// Tek sorguda toplanıyor: oturum listesi gösterilirken her satır için ayrı
// bir puan sorgusu, yirmi oturumluk bir sayfada yirmi bir tur demek olurdu.
func (r *ScoreRepo) ListBySessions(ctx context.Context, scope repository.Scope, sessionIDs []uuid.UUID) (map[uuid.UUID]*model.Score, error) {
	if len(sessionIDs) == 0 {
		return map[uuid.UUID]*model.Score{}, nil
	}
	condition, err := scopedFilter(scope, bson.M{"session_id": bson.M{"$in": sessionIDs}})
	if err != nil {
		return nil, err
	}
	cursor, err := r.coll.Find(ctx, condition)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "puanlar listelenemedi", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	scores := make(map[uuid.UUID]*model.Score, len(sessionIDs))
	for cursor.Next(ctx) {
		var s model.Score
		if err := cursor.Decode(&s); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "puan çözümlenemedi", err)
		}
		scores[s.SessionID] = &s
	}
	if err := cursor.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "puanlar listelenemedi", err)
	}
	return scores, nil
}

// ApplyOverride writes the manager's override on top of the machine score.
//
// Önceki değerler ezme kaydının içinde saklanıyor, silinmiyor: ezilmiş bir
// puanın makine sonucu görülemezse, ezme yetkisi denetlenemez bir yetkiye
// dönüşür.
func (r *ScoreRepo) ApplyOverride(
	ctx context.Context,
	scope repository.Scope,
	scoreID uuid.UUID,
	total float64,
	passed bool,
	override model.ScoreOverride,
) (*model.Score, error) {
	condition, err := scopedFilter(scope, bson.M{"_id": scoreID})
	if err != nil {
		return nil, err
	}
	if override.OverriddenAt.IsZero() {
		override.OverriddenAt = r.now().UTC()
	}

	var updated model.Score
	err = r.coll.FindOneAndUpdate(ctx, condition,
		bson.M{"$set": bson.M{"total": total, "passed": passed, "override": override}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domainErr.New(domainErr.ErrNotFound, "puan bulunamadı", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "puan ezilemedi", err)
	}
	return &updated, nil
}

var _ repository.ScoreRepository = (*ScoreRepo)(nil)
