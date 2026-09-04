// Package prova, Prova belgelerinin MongoDB uygulamasıdır.
package prova

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/prova/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// defaultListLimit, listeleme sorgularının varsayılan tavanı.
const defaultListLimit = 100

// maxListLimit, istemcinin isteyebileceği en yüksek sayfa boyutu.
const maxListLimit = 500

// versionedDoc, sürümlenebilir belgelerin taşıması gereken sözleşme.
//
// Pointer kısıtı (*T) sayesinde depo, belgenin zarfına yazabiliyor: sürüm
// numarasını ve durumu değiştirmek zarfı değiştirmek demek.
type versionedDoc[T any] interface {
	*T
	Envelope() *model.Document
}

// VersionedRepo, sürümlenebilir belgeler için ortak MongoDB uygulaması.
//
// Sürümleme kuralları burada, tek yerde zorlanıyor. Dört ayrı belge tipinin
// her birine ayrı yazılsaydı, dördü zamanla ayrışır ve "yayınlanmış belge
// değişmez" kuralı en az birinde delinirdi.
type VersionedRepo[T any, PT versionedDoc[T]] struct {
	coll *mongo.Collection
	now  func() time.Time
}

// NewVersionedRepo wires a repository over one collection.
func NewVersionedRepo[T any, PT versionedDoc[T]](db *mongo.Database, collection string) *VersionedRepo[T, PT] {
	return &VersionedRepo[T, PT]{coll: db.Collection(collection), now: time.Now}
}

// filter, kiracı sınırını uygulayarak sorgu kurar.
//
// org_id en sona yazılıyor: çağıranın kendi "org_id" koşulu — ister hata,
// ister istemciden gelen bir değer — sınırı genişletemesin.
func (r *VersionedRepo[T, PT]) filter(scope repository.Scope, conditions bson.M) (bson.M, error) {
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

// Create, yeni bir soyun ilk taslağını yazar.
func (r *VersionedRepo[T, PT]) Create(ctx context.Context, scope repository.Scope, doc *T) error {
	envelope := PT(doc).Envelope()
	if scope.IsZero() {
		return model.ErrOrgRequired
	}
	// Kiracı, çağıranın gönderdiği belgeden değil scope'tan alınır. Belgeden
	// alsaydık, başka bir organizasyonun kimliğini taşıyan bir gövde
	// kiracı sınırını delerdi.
	envelope.OrgID = scope.OrgID()
	if envelope.ID == uuid.Nil {
		envelope.ID = uuid.New()
	}
	if envelope.LineageID == uuid.Nil {
		envelope.LineageID = envelope.ID
	}
	if envelope.Version == 0 {
		envelope.Version = 1
	}
	if envelope.Status == "" {
		envelope.Status = model.DocumentStatusDraft
	}
	if envelope.CreatedAt.IsZero() {
		envelope.CreatedAt = r.now().UTC()
	}

	if _, err := r.coll.InsertOne(ctx, doc); err != nil {
		return wrapWrite(err, "belge oluşturulamadı")
	}
	return nil
}

// UpdateDraft, taslağı yerinde günceller ve yayınlanmışa dokunmayı reddeder.
func (r *VersionedRepo[T, PT]) UpdateDraft(ctx context.Context, scope repository.Scope, doc *T) error {
	envelope := PT(doc).Envelope()

	// Durum koşulu sorgunun içinde: önce okuyup sonra yazan bir kontrol,
	// iki eşzamanlı isteğin arasına yayınlama girdiğinde yanlış cevap verir.
	condition, err := r.filter(scope, bson.M{
		"_id":    envelope.ID,
		"status": model.DocumentStatusDraft,
	})
	if err != nil {
		return err
	}

	envelope.OrgID = scope.OrgID()
	result, err := r.coll.ReplaceOne(ctx, condition, doc)
	if err != nil {
		return wrapWrite(err, "taslak güncellenemedi")
	}
	if result.MatchedCount == 0 {
		// Eşleşme yoksa iki olasılık var: belge yok, ya da taslak değil.
		// İkisini ayırt etmek çağırana anlamlı bir cevap vermenin tek yolu.
		return r.explainMissingDraft(ctx, scope, envelope.ID)
	}
	return nil
}

// explainMissingDraft, eşleşmeyen bir güncellemenin nedenini söyler.
func (r *VersionedRepo[T, PT]) explainMissingDraft(ctx context.Context, scope repository.Scope, id uuid.UUID) error {
	condition, err := r.filter(scope, bson.M{"_id": id})
	if err != nil {
		return err
	}
	count, err := r.coll.CountDocuments(ctx, condition)
	if err != nil {
		return wrapWrite(err, "belge durumu okunamadı")
	}
	if count == 0 {
		return model.ErrDocumentNotFound
	}
	return model.ErrDocumentFrozen
}

// NewVersion, son sürümden yeni bir taslak türetir.
//
// apply, yeni taslağın gövdesini düzenler. Fonksiyon olarak alınıyor çünkü
// her belge tipinin düzenlenecek alanları farklı, ama sürüm zarfının nasıl
// ilerleyeceği aynı.
func (r *VersionedRepo[T, PT]) NewVersion(
	ctx context.Context,
	scope repository.Scope,
	lineageID, createdBy uuid.UUID,
	apply func(*T),
) (*T, error) {
	latest, err := r.GetLatest(ctx, scope, lineageID)
	if err != nil {
		return nil, err
	}

	next := *latest
	envelope := PT(&next).Envelope()
	*envelope = PT(latest).Envelope().NextVersion(createdBy, r.now())

	if apply != nil {
		apply(&next)
	}
	// Zarf apply'dan sonra tekrar sabitleniyor: çağıranın gövdeyi düzenlerken
	// sürüm numarasını ya da durumu değiştirmesi, sürümlemenin tamamını
	// bozardı.
	envelope.OrgID = scope.OrgID()
	envelope.Status = model.DocumentStatusDraft
	envelope.PublishedAt = nil
	envelope.ArchivedAt = nil

	if _, err := r.coll.InsertOne(ctx, &next); err != nil {
		return nil, wrapWrite(err, "yeni sürüm oluşturulamadı")
	}
	return &next, nil
}

// GetLatest, soyun en yüksek sürüm numaralı belgesini döndürür.
func (r *VersionedRepo[T, PT]) GetLatest(ctx context.Context, scope repository.Scope, lineageID uuid.UUID) (*T, error) {
	return r.findOne(ctx, scope, bson.M{"lineage_id": lineageID},
		options.FindOne().SetSort(bson.D{{Key: "version", Value: -1}}))
}

// GetLatestPublished, soyun oynanabilir son sürümünü döndürür.
func (r *VersionedRepo[T, PT]) GetLatestPublished(ctx context.Context, scope repository.Scope, lineageID uuid.UUID) (*T, error) {
	return r.findOne(ctx, scope,
		bson.M{"lineage_id": lineageID, "status": model.DocumentStatusPublished},
		options.FindOne().SetSort(bson.D{{Key: "version", Value: -1}}))
}

// GetVersion, belirli bir sürümü döndürür.
func (r *VersionedRepo[T, PT]) GetVersion(ctx context.Context, scope repository.Scope, lineageID uuid.UUID, version int) (*T, error) {
	return r.findOne(ctx, scope, bson.M{"lineage_id": lineageID, "version": version}, nil)
}

// GetByID, sürüm kimliğiyle tek belge döndürür.
func (r *VersionedRepo[T, PT]) GetByID(ctx context.Context, scope repository.Scope, versionID uuid.UUID) (*T, error) {
	return r.findOne(ctx, scope, bson.M{"_id": versionID}, nil)
}

func (r *VersionedRepo[T, PT]) findOne(ctx context.Context, scope repository.Scope, conditions bson.M, opts *options.FindOneOptionsBuilder) (*T, error) {
	condition, err := r.filter(scope, conditions)
	if err != nil {
		return nil, err
	}

	var result T
	var single *mongo.SingleResult
	if opts != nil {
		single = r.coll.FindOne(ctx, condition, opts)
	} else {
		single = r.coll.FindOne(ctx, condition)
	}
	if err := single.Decode(&result); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, model.ErrDocumentNotFound
		}
		return nil, domainErr.New(domainErr.ErrInternal, "belge okunamadı", err)
	}
	return &result, nil
}

// List, soy başına son sürümleri döndürür.
//
// Toplama veritabanında yapılıyor: tüm sürümleri çekip Go'da soya göre
// gruplamak, on sürümlü bir senaryonun listede on kez görünmemesi için
// koleksiyonun tamamını belleğe almak demek olurdu.
func (r *VersionedRepo[T, PT]) List(ctx context.Context, scope repository.Scope, opts repository.ListOptions) ([]*T, error) {
	match := bson.M{}
	if opts.PublishedOnly {
		match["status"] = model.DocumentStatusPublished
	}
	condition, err := r.filter(scope, match)
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

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: condition}},
		{{Key: "$sort", Value: bson.D{{Key: "version", Value: -1}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$lineage_id"},
			{Key: "doc", Value: bson.D{{Key: "$first", Value: "$$ROOT"}}},
		}}},
		{{Key: "$replaceRoot", Value: bson.D{{Key: "newRoot", Value: "$doc"}}}},
		{{Key: "$sort", Value: bson.D{{Key: "created_at", Value: -1}}}},
		{{Key: "$skip", Value: int64(max(opts.Offset, 0))}},
		{{Key: "$limit", Value: int64(limit)}},
	}

	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "belgeler listelenemedi", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []*T
	for cursor.Next(ctx) {
		var item T
		if err := cursor.Decode(&item); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "belge çözümlenemedi", err)
		}
		docs = append(docs, &item)
	}
	if err := cursor.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "belgeler listelenemedi", err)
	}
	return docs, nil
}

// Publish, taslağı oynanabilir hâle getirir.
func (r *VersionedRepo[T, PT]) Publish(ctx context.Context, scope repository.Scope, lineageID uuid.UUID, version int) (*T, error) {
	return r.transition(ctx, scope, lineageID, version,
		model.DocumentStatusDraft, model.DocumentStatusPublished, "published_at")
}

// Archive, sürümü yeni oturumlardan çeker.
//
// Yalnızca yayınlanmış sürüm arşivlenebilir: bir taslağı arşivlemek, hiç
// yayınlanmamış bir metni "geri çekmek" gibi anlamsız bir işlem olurdu.
func (r *VersionedRepo[T, PT]) Archive(ctx context.Context, scope repository.Scope, lineageID uuid.UUID, version int) (*T, error) {
	return r.transition(ctx, scope, lineageID, version,
		model.DocumentStatusPublished, model.DocumentStatusArchived, "archived_at")
}

func (r *VersionedRepo[T, PT]) transition(
	ctx context.Context,
	scope repository.Scope,
	lineageID uuid.UUID,
	version int,
	from, to model.DocumentStatus,
	stampField string,
) (*T, error) {
	condition, err := r.filter(scope, bson.M{
		"lineage_id": lineageID,
		"version":    version,
		"status":     from,
	})
	if err != nil {
		return nil, err
	}

	update := bson.M{"$set": bson.M{"status": to, stampField: r.now().UTC()}}

	var result T
	err = r.coll.FindOneAndUpdate(ctx, condition, update,
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, r.explainFailedTransition(ctx, scope, lineageID, version)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "belge durumu değiştirilemedi", err)
	}
	return &result, nil
}

func (r *VersionedRepo[T, PT]) explainFailedTransition(ctx context.Context, scope repository.Scope, lineageID uuid.UUID, version int) error {
	condition, err := r.filter(scope, bson.M{"lineage_id": lineageID, "version": version})
	if err != nil {
		return err
	}
	count, err := r.coll.CountDocuments(ctx, condition)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "belge durumu okunamadı", err)
	}
	if count == 0 {
		return model.ErrDocumentNotFound
	}
	return model.ErrDocumentNotDraft
}

// wrapWrite, sürücü hatasını domain hatasına çevirir.
//
// Benzersizlik ihlali ayrı ele alınıyor: (org_id, lineage_id, version)
// index'i, iki eşzamanlı yayının aynı sürüm numarasını üretmesini engelleyen
// tek şey, ve o çarpışma çağırana "çakışma" olarak dönmeli.
func wrapWrite(err error, message string) error {
	if mongo.IsDuplicateKeyError(err) {
		return domainErr.New(domainErr.ErrConflict,
			"bu sürüm numarası zaten kullanılıyor; belge eşzamanlı olarak değiştirilmiş olabilir", err)
	}
	return domainErr.New(domainErr.ErrInternal, fmt.Sprintf("%s", message), err)
}
