package prova

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/prova/repository"
	provaMongo "github.com/masterfabric-go/masterfabric/internal/infrastructure/mongo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// testDB, çalışan bir MongoDB'ye bağlanır ya da testi atlar.
//
// Sürümleme kurallarının çoğu sorgunun içinde yaşıyor ("yalnızca taslak
// eşleşsin", "aynı sürüm numarası iki kez yazılamasın"). Sahte bir depoya
// karşı test etmek, tam da doğrulanması gereken şeyi test dışında bırakırdı.
func testDB(t *testing.T) *mongo.Database {
	t.Helper()

	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().
		ApplyURI(uri).
		SetConnectTimeout(2 * time.Second).
		SetServerSelectionTimeout(2 * time.Second).
		SetRegistry(provaMongo.NewRegistry()))
	if err != nil {
		t.Skipf("MongoDB kullanılamıyor, sürümleme testleri atlanıyor: %v", err)
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		t.Skipf("MongoDB kullanılamıyor, sürümleme testleri atlanıyor: %v", err)
	}

	name := "prova_test_" + uuid.NewString()[:8]
	db := client.Database(name)
	t.Cleanup(func() {
		_ = db.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})

	if err := provaMongo.EnsureIndexes(ctx, db); err != nil {
		t.Fatalf("index'ler oluşturulamadı: %v", err)
	}
	return db
}

func newCharacter(orgID, author uuid.UUID, name string) *model.Character {
	c := model.NewCharacter(orgID, author, time.Now)
	c.Name = name
	c.Persona = "Sabırsız bir müşteri."
	c.BehaviorRules = []string{"Konuyu dağıtır."}
	c.Difficulty = model.DifficultyMedium
	return c
}

// Faz 2'nin ilk kontrol maddesi: düzenleme v2 üretir ve v1 hâlâ okunabilir.
func TestVersionedRepo_EditCreatesNewVersionAndKeepsTheOld(t *testing.T) {
	ctx := context.Background()
	repo := NewCharacterRepo(testDB(t))
	orgID, author := uuid.New(), uuid.New()
	scope := repository.NewScope(orgID)

	original := newCharacter(orgID, author, "Ayşe")
	if err := repo.Create(ctx, scope, original); err != nil {
		t.Fatalf("oluşturma başarısız: %v", err)
	}
	if _, err := repo.Publish(ctx, scope, original.LineageID, 1); err != nil {
		t.Fatalf("yayınlama başarısız: %v", err)
	}

	v2, err := repo.NewVersion(ctx, scope, original.LineageID, author, func(c *model.Character) {
		c.Name = "Ayşe (zorlu)"
		c.Difficulty = model.DifficultyHigh
	})
	if err != nil {
		t.Fatalf("yeni sürüm başarısız: %v", err)
	}

	if v2.Version != 2 {
		t.Fatalf("sürüm 2 bekleniyordu, %d geldi", v2.Version)
	}
	if v2.LineageID != original.LineageID {
		t.Fatal("soy kimliği sürümler arasında sabit kalmalı")
	}
	if v2.Status != model.DocumentStatusDraft {
		t.Fatalf("yeni sürüm taslak olarak başlamalı, %q geldi", v2.Status)
	}

	// Asıl iddia: eski sürüm olduğu gibi duruyor.
	v1, err := repo.GetVersion(ctx, scope, original.LineageID, 1)
	if err != nil {
		t.Fatalf("v1 okunamadı: %v", err)
	}
	if v1.Name != "Ayşe" || v1.Difficulty != model.DifficultyMedium {
		t.Fatalf("v1 değişmiş olmamalı: %+v", v1)
	}
	if v1.Status != model.DocumentStatusPublished {
		t.Fatal("v1 yayınlanmış kalmalı")
	}
}

// İkinci kontrol maddesi: yayınlanmış belgeye update denemesi hata verir.
func TestVersionedRepo_RefusesToUpdateAPublishedDocument(t *testing.T) {
	ctx := context.Background()
	repo := NewCharacterRepo(testDB(t))
	orgID, author := uuid.New(), uuid.New()
	scope := repository.NewScope(orgID)

	doc := newCharacter(orgID, author, "Mehmet")
	if err := repo.Create(ctx, scope, doc); err != nil {
		t.Fatalf("oluşturma başarısız: %v", err)
	}

	// Taslakken güncelleme serbest.
	doc.Name = "Mehmet (taslak düzeltmesi)"
	if err := repo.UpdateDraft(ctx, scope, doc); err != nil {
		t.Fatalf("taslak güncellenebilmeliydi: %v", err)
	}

	published, err := repo.Publish(ctx, scope, doc.LineageID, 1)
	if err != nil {
		t.Fatalf("yayınlama başarısız: %v", err)
	}

	published.Name = "gizlice değiştirildi"
	err = repo.UpdateDraft(ctx, scope, published)
	if !errors.Is(err, model.ErrDocumentFrozen) {
		t.Fatalf("yayınlanmış belge güncellenememeli, gelen hata: %v", err)
	}

	// Ve gerçekten değişmemiş olmalı.
	stored, err := repo.GetVersion(ctx, scope, doc.LineageID, 1)
	if err != nil {
		t.Fatalf("okuma başarısız: %v", err)
	}
	if stored.Name == "gizlice değiştirildi" {
		t.Fatal("reddedilen güncelleme yine de yazılmış")
	}
}

// Kiracı sınırı, unutulabilecek bir filtre değil; sorgu kurulamamalı.
func TestVersionedRepo_RefusesQueriesWithoutAnOrganisation(t *testing.T) {
	ctx := context.Background()
	repo := NewCharacterRepo(testDB(t))
	empty := repository.Scope{}

	if err := repo.Create(ctx, empty, newCharacter(uuid.New(), uuid.New(), "X")); !errors.Is(err, model.ErrOrgRequired) {
		t.Fatalf("org'suz oluşturma reddedilmeli: %v", err)
	}
	if _, err := repo.GetLatest(ctx, empty, uuid.New()); !errors.Is(err, model.ErrOrgRequired) {
		t.Fatalf("org'suz okuma reddedilmeli: %v", err)
	}
	if _, err := repo.List(ctx, empty, repository.ListOptions{}); !errors.Is(err, model.ErrOrgRequired) {
		t.Fatalf("org'suz listeleme reddedilmeli: %v", err)
	}
}

// Bir organizasyonun belgesi başka bir organizasyondan görünmemeli. Bu,
// uygulama kodunda unutulan bir filtreye değil, depo tipine bağlı olmalı.
func TestVersionedRepo_IsolatesTenants(t *testing.T) {
	ctx := context.Background()
	repo := NewCharacterRepo(testDB(t))
	author := uuid.New()
	orgA, orgB := repository.NewScope(uuid.New()), repository.NewScope(uuid.New())

	doc := newCharacter(orgA.OrgID(), author, "Gizli")
	if err := repo.Create(ctx, orgA, doc); err != nil {
		t.Fatalf("oluşturma başarısız: %v", err)
	}

	if _, err := repo.GetLatest(ctx, orgB, doc.LineageID); !errors.Is(err, model.ErrDocumentNotFound) {
		t.Fatalf("başka kiracının belgesi görünmemeli: %v", err)
	}

	list, err := repo.List(ctx, orgB, repository.ListOptions{})
	if err != nil {
		t.Fatalf("listeleme başarısız: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("başka kiracının listesi boş olmalı, %d belge geldi", len(list))
	}
}

// Belgeyi gönderen taraf başka bir organizasyonun kimliğini yazsa bile,
// kiracı scope'tan alınmalı.
func TestVersionedRepo_IgnoresOrgIDFromTheDocumentBody(t *testing.T) {
	ctx := context.Background()
	repo := NewCharacterRepo(testDB(t))
	owner := repository.NewScope(uuid.New())
	intruder := uuid.New()

	doc := newCharacter(intruder, uuid.New(), "Sahte")
	if err := repo.Create(ctx, owner, doc); err != nil {
		t.Fatalf("oluşturma başarısız: %v", err)
	}

	if doc.OrgID != owner.OrgID() {
		t.Fatalf("kiracı scope'tan alınmalı, gövdeden değil: %s", doc.OrgID)
	}
	if _, err := repo.GetLatest(ctx, repository.NewScope(intruder), doc.LineageID); !errors.Is(err, model.ErrDocumentNotFound) {
		t.Fatal("gövdedeki organizasyon kimliği sınırı delmemeli")
	}
}

// Listeleme soy başına tek satır döndürmeli: on sürümlü bir karakter listede
// on kez görünürse ekran kullanılamaz hâle gelir.
func TestVersionedRepo_ListReturnsOneRowPerLineage(t *testing.T) {
	ctx := context.Background()
	repo := NewCharacterRepo(testDB(t))
	author := uuid.New()
	scope := repository.NewScope(uuid.New())

	doc := newCharacter(scope.OrgID(), author, "Çok sürümlü")
	if err := repo.Create(ctx, scope, doc); err != nil {
		t.Fatalf("oluşturma başarısız: %v", err)
	}
	for version := 1; version <= 3; version++ {
		if _, err := repo.Publish(ctx, scope, doc.LineageID, version); err != nil {
			t.Fatalf("v%d yayınlanamadı: %v", version, err)
		}
		if _, err := repo.NewVersion(ctx, scope, doc.LineageID, author, nil); err != nil {
			t.Fatalf("v%d sonrası yeni sürüm oluşturulamadı: %v", version, err)
		}
	}

	list, err := repo.List(ctx, scope, repository.ListOptions{})
	if err != nil {
		t.Fatalf("listeleme başarısız: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("soy başına tek satır beklenir, %d geldi", len(list))
	}
	if list[0].Version != 4 {
		t.Fatalf("listede en yüksek sürüm görünmeli, %d geldi", list[0].Version)
	}
}

// publishedOnly, henüz yayınlanmamış taslakları gizlemeli: çalışan, üzerinde
// çalışılan bir senaryoyu oynayabilmemeli.
func TestVersionedRepo_ListPublishedOnlyHidesDrafts(t *testing.T) {
	ctx := context.Background()
	repo := NewCharacterRepo(testDB(t))
	scope := repository.NewScope(uuid.New())

	draft := newCharacter(scope.OrgID(), uuid.New(), "Taslak")
	if err := repo.Create(ctx, scope, draft); err != nil {
		t.Fatalf("oluşturma başarısız: %v", err)
	}

	list, err := repo.List(ctx, scope, repository.ListOptions{PublishedOnly: true})
	if err != nil {
		t.Fatalf("listeleme başarısız: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("yayınlanmamış taslak listede görünmemeli, %d geldi", len(list))
	}
}

// Aynı sürüm numarasının iki kez yazılması, benzersizlik index'iyle
// engellenmeli: "sürüm 3" iki farklı belgeyi adlandırırsa, ona atıfta bulunan
// her sertifika belirsizleşir.
func TestVersionedRepo_RejectsDuplicateVersionNumbers(t *testing.T) {
	ctx := context.Background()
	repo := NewCharacterRepo(testDB(t))
	scope := repository.NewScope(uuid.New())

	first := newCharacter(scope.OrgID(), uuid.New(), "Bir")
	if err := repo.Create(ctx, scope, first); err != nil {
		t.Fatalf("oluşturma başarısız: %v", err)
	}

	clash := newCharacter(scope.OrgID(), uuid.New(), "İki")
	clash.ID = uuid.New()
	clash.LineageID = first.LineageID
	clash.Version = 1

	if err := repo.Create(ctx, scope, clash); err == nil {
		t.Fatal("aynı soyda aynı sürüm numarası iki kez yazılamamalı")
	}
}

// Yalnızca taslak yayınlanabilir; ikinci yayınlama denemesi çakışma dönmeli.
func TestVersionedRepo_PublishIsIdempotentlyRefusedOnPublishedVersions(t *testing.T) {
	ctx := context.Background()
	repo := NewCharacterRepo(testDB(t))
	scope := repository.NewScope(uuid.New())

	doc := newCharacter(scope.OrgID(), uuid.New(), "Tek")
	if err := repo.Create(ctx, scope, doc); err != nil {
		t.Fatalf("oluşturma başarısız: %v", err)
	}
	if _, err := repo.Publish(ctx, scope, doc.LineageID, 1); err != nil {
		t.Fatalf("ilk yayınlama başarısız: %v", err)
	}
	if _, err := repo.Publish(ctx, scope, doc.LineageID, 1); !errors.Is(err, model.ErrDocumentNotDraft) {
		t.Fatalf("ikinci yayınlama reddedilmeli: %v", err)
	}
}

// Arşivleme yeni oturumları engeller ama mevcut referansları bozmaz:
// arşivlenmiş bir sürüm sürüm kimliğiyle hâlâ okunabilmeli.
func TestVersionedRepo_ArchivedVersionsStayReadable(t *testing.T) {
	ctx := context.Background()
	repo := NewCharacterRepo(testDB(t))
	scope := repository.NewScope(uuid.New())

	doc := newCharacter(scope.OrgID(), uuid.New(), "Emekli")
	if err := repo.Create(ctx, scope, doc); err != nil {
		t.Fatalf("oluşturma başarısız: %v", err)
	}
	if _, err := repo.Publish(ctx, scope, doc.LineageID, 1); err != nil {
		t.Fatalf("yayınlama başarısız: %v", err)
	}
	archived, err := repo.Archive(ctx, scope, doc.LineageID, 1)
	if err != nil {
		t.Fatalf("arşivleme başarısız: %v", err)
	}
	if archived.ArchivedAt == nil {
		t.Fatal("arşivleme zamanı damgalanmalı")
	}

	if _, err := repo.GetByID(ctx, scope, doc.ID); err != nil {
		t.Fatalf("arşivlenmiş sürüm sürüm kimliğiyle okunabilmeli: %v", err)
	}
	if _, err := repo.GetLatestPublished(ctx, scope, doc.LineageID); !errors.Is(err, model.ErrDocumentNotFound) {
		t.Fatal("arşivlenmiş sürüm oynanabilir sayılmamalı")
	}
}

// UUID'ler BSON binary olarak yazılmalı; dizi olarak yazılırlarsa başka
// dildeki bir istemci aynı belgeyi UUID olarak göremez.
func TestVersionedRepo_StoresUUIDsAsBSONBinary(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	repo := NewCharacterRepo(db)
	scope := repository.NewScope(uuid.New())

	doc := newCharacter(scope.OrgID(), uuid.New(), "Kodek")
	if err := repo.Create(ctx, scope, doc); err != nil {
		t.Fatalf("oluşturma başarısız: %v", err)
	}

	// Ham hâliyle okunuyor: any'ye çözmek kodeği atlar ve depodaki gerçek
	// BSON tipini gösterir.
	var raw struct {
		ID    any `bson:"_id"`
		OrgID any `bson:"org_id"`
	}
	if err := db.Collection(provaMongo.CollectionCharacters).
		FindOne(ctx, map[string]any{"_id": doc.ID}).Decode(&raw); err != nil {
		t.Fatalf("ham belge okunamadı: %v", err)
	}

	for name, value := range map[string]any{"_id": raw.ID, "org_id": raw.OrgID} {
		binary, ok := value.(bson.Binary)
		if !ok {
			t.Fatalf("%s BSON binary olmalı, %T geldi", name, value)
		}
		if binary.Subtype != bson.TypeBinaryUUID {
			t.Fatalf("%s subtype 4 (UUID) olmalı, %d geldi", name, binary.Subtype)
		}
		if len(binary.Data) != 16 {
			t.Fatalf("%s 16 bayt olmalı, %d geldi", name, len(binary.Data))
		}
	}

	// Ve kodek üzerinden geri okuduğumuzda aynı kimliğe dönmeli.
	stored, err := repo.GetByID(ctx, scope, doc.ID)
	if err != nil {
		t.Fatalf("kodek üzerinden okuma başarısız: %v", err)
	}
	if stored.ID != doc.ID || stored.OrgID != scope.OrgID() {
		t.Fatal("UUID gidiş dönüşte korunmalı")
	}
}
