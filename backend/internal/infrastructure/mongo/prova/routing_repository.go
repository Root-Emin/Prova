package prova

import (
	"context"
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

// RoutingRepo, yönlendirme kararları.
type RoutingRepo struct {
	coll *mongo.Collection
	now  func() time.Time
}

// NewRoutingRepo wires the repository.
func NewRoutingRepo(db *mongo.Database) *RoutingRepo {
	return &RoutingRepo{coll: db.Collection(provaMongo.CollectionRoutingRecord), now: time.Now}
}

// Record writes one routing decision.
func (r *RoutingRepo) Record(ctx context.Context, scope repository.Scope, record *model.RoutingRecord) error {
	if scope.IsZero() {
		return model.ErrOrgRequired
	}
	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}
	record.OrgID = scope.OrgID()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = r.now().UTC()
	}
	if _, err := r.coll.InsertOne(ctx, record); err != nil {
		return domainErr.New(domainErr.ErrInternal, "yönlendirme kaydı yazılamadı", err)
	}
	return nil
}

// List returns routing records, newest first.
func (r *RoutingRepo) List(ctx context.Context, scope repository.Scope, sessionID *uuid.UUID, limit int) ([]*model.RoutingRecord, error) {
	conditions := bson.M{}
	if sessionID != nil {
		conditions["session_id"] = *sessionID
	}
	condition, err := scopedFilter(scope, conditions)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	cursor, err := r.coll.Find(ctx, condition,
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "yönlendirme kayıtları listelenemedi", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var records []*model.RoutingRecord
	for cursor.Next(ctx) {
		var rec model.RoutingRecord
		if err := cursor.Decode(&rec); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "yönlendirme kaydı çözümlenemedi", err)
		}
		records = append(records, &rec)
	}
	if err := cursor.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "yönlendirme kayıtları listelenemedi", err)
	}
	return records, nil
}

// Stats aggregates per-tier usage and counts failovers.
//
// Toplama veritabanında yapılıyor: bir yılın çağrı kayıtlarını çekip Go'da
// toplamak, istatistik sorgusunu kullanılamaz hâle getirirdi.
func (r *RoutingRepo) Stats(ctx context.Context, scope repository.Scope, from, to *time.Time) ([]model.TierStats, int, error) {
	conditions := bson.M{}
	if from != nil || to != nil {
		window := bson.M{}
		if from != nil {
			window["$gte"] = from.UTC()
		}
		if to != nil {
			window["$lte"] = to.UTC()
		}
		conditions["created_at"] = window
	}
	condition, err := scopedFilter(scope, conditions)
	if err != nil {
		return nil, 0, err
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: condition}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$tier"},
			{Key: "calls", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "avg_latency_ms", Value: bson.D{{Key: "$avg", Value: "$latency_ms"}}},
			{Key: "cost_usd", Value: bson.D{{Key: "$sum", Value: "$cost_usd"}}},
			{Key: "input_tokens", Value: bson.D{{Key: "$sum", Value: "$input_tokens"}}},
			{Key: "output_tokens", Value: bson.D{{Key: "$sum", Value: "$output_tokens"}}},
			{Key: "failovers", Value: bson.D{{Key: "$sum", Value: bson.D{
				{Key: "$cond", Value: bson.A{
					bson.D{{Key: "$eq", Value: bson.A{"$rule", model.RuleFailover}}}, 1, 0,
				}},
			}}}},
		}}},
	}

	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "yönlendirme istatistiği hesaplanamadı", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var (
		stats     []model.TierStats
		failovers int
	)
	for cursor.Next(ctx) {
		var row struct {
			Tier         model.Tier `bson:"_id"`
			Calls        int        `bson:"calls"`
			AvgLatencyMs float64    `bson:"avg_latency_ms"`
			CostUSD      float64    `bson:"cost_usd"`
			InputTokens  int        `bson:"input_tokens"`
			OutputTokens int        `bson:"output_tokens"`
			Failovers    int        `bson:"failovers"`
		}
		if err := cursor.Decode(&row); err != nil {
			return nil, 0, domainErr.New(domainErr.ErrInternal, "istatistik satırı çözümlenemedi", err)
		}
		stats = append(stats, model.TierStats{
			Tier:         row.Tier,
			Calls:        row.Calls,
			AvgLatencyMs: row.AvgLatencyMs,
			CostUSD:      row.CostUSD,
			InputTokens:  row.InputTokens,
			OutputTokens: row.OutputTokens,
		})
		failovers += row.Failovers
	}
	if err := cursor.Err(); err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "yönlendirme istatistiği hesaplanamadı", err)
	}
	return stats, failovers, nil
}

var _ repository.RoutingRepository = (*RoutingRepo)(nil)
