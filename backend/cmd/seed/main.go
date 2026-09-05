// Command seed, geliştirme ve doğrulama için tohum verisi yazar.
//
// Çalıştırma: go run ./cmd/seed
//
// Betik yeniden çalıştırılabilir: her varlık sabit bir kimlikle yazılır ve
// mevcutsa üzerine yazılmaz. Doğrulama betiği bu kimliklere dayanıyor, ve
// her çalıştırmada yeni kimlik üretmek onları kullanılamaz hâle getirirdi.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	iamModel "github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	tenantModel "github.com/masterfabric-go/masterfabric/internal/domain/tenant/model"
	pgIam "github.com/masterfabric-go/masterfabric/internal/infrastructure/postgres/iam"
	pgTenant "github.com/masterfabric-go/masterfabric/internal/infrastructure/postgres/tenant"
	"github.com/masterfabric-go/masterfabric/internal/shared/config"
	"github.com/masterfabric-go/masterfabric/internal/shared/database"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// Sabit kimlikler. Doğrulama ve duman testi betikleri bunlara atıfta bulunuyor.
var (
	demoOrgID     = uuid.MustParse("11111111-1111-4111-8111-111111111111")
	adminUserID   = uuid.MustParse("22222222-2222-4222-8222-222222222222")
	traineeUserID = uuid.MustParse("33333333-3333-4333-8333-333333333333")

	calmCharacterID   = uuid.MustParse("44444444-4444-4444-8444-444444444444")
	hostileCharacterD = uuid.MustParse("55555555-5555-4555-8555-555555555555")
	serviceRubricID   = uuid.MustParse("66666666-6666-4666-8666-666666666666")
	refundScenarioID  = uuid.MustParse("77777777-7777-4777-8777-777777777777")
	kvkkScenarioID    = uuid.MustParse("88888888-8888-4888-8888-888888888888")
	fastProfileID     = uuid.MustParse("99999999-9999-4999-8999-999999999999")
	strongProfileID   = uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
)

// Tohum kullanıcılarının adresleri.
const (
	adminEmail   = "yonetici@prova.local"
	traineeEmail = "calisan@prova.local"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "tohumlama başarısız: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	db, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	defer db.Close()

	mongoDB, err := database.NewMongoClient(ctx, cfg.Mongo)
	if err != nil {
		return fmt.Errorf("mongodb: %w", err)
	}
	defer func() { _ = mongoDB.Close(context.Background()) }()

	log.Println("🌱 Prova tohum verisi yazılıyor...")

	if err := seedIdentity(ctx, db); err != nil {
		return err
	}
	if err := seedContent(ctx, mongoDB); err != nil {
		return err
	}

	log.Println("✅ Tohumlama tamamlandı")
	log.Printf("   organizasyon: %s", demoOrgID)
	log.Printf("   yönetici    : %s (%s)", adminEmail, adminUserID)
	log.Printf("   çalışan     : %s (%s)", traineeEmail, traineeUserID)
	return nil
}

// seedIdentity, PostgreSQL tarafını yazar: organizasyon, kullanıcılar,
// roller ve izin eşlemeleri.
func seedIdentity(ctx context.Context, db *pgxpool.Pool) error {
	orgs := pgTenant.NewOrgRepo(db)
	users := pgIam.NewUserRepo(db)
	orgUsers := pgIam.NewOrgUserRepo(db)
	roles := pgIam.NewRoleRepo(db)

	org := &tenantModel.Organization{
		ID:     demoOrgID,
		Name:   "Prova Demo Organizasyonu",
		Slug:   "prova-demo",
		Status: tenantModel.OrgStatusActive,
	}
	if err := createIfMissing(ctx, org.ID, func() error { return orgs.Create(ctx, org) },
		func() error { _, err := orgs.GetByID(ctx, org.ID); return err }); err != nil {
		return fmt.Errorf("organizasyon: %w", err)
	}

	now := time.Now().UTC()
	seedUsers := []*iamModel.User{
		{ID: adminUserID, Email: adminEmail, FirstName: "Demo", LastName: "Yönetici",
			Status: iamModel.UserStatusActive, EmailVerifiedAt: &now},
		{ID: traineeUserID, Email: traineeEmail, FirstName: "Demo", LastName: "Çalışan",
			Status: iamModel.UserStatusActive, EmailVerifiedAt: &now},
	}
	for _, user := range seedUsers {
		if err := createIfMissing(ctx, user.ID, func() error { return users.Create(ctx, user) },
			func() error { _, err := users.GetByID(ctx, user.ID); return err }); err != nil {
			return fmt.Errorf("kullanıcı %s: %w", user.Email, err)
		}
		if err := orgUsers.Add(ctx, &iamModel.OrganizationUser{
			OrganizationID: demoOrgID,
			UserID:         user.ID,
			Status:         iamModel.OrgUserStatusActive,
		}); err != nil {
			return fmt.Errorf("üyelik %s: %w", user.Email, err)
		}
	}

	// Roller ve izinler.
	//
	// Yönetici rolünde joker kullanılmıyor: izinler tek tek sayılıyor, çünkü
	// bir yönetici rolünün neye eriştiği okunabilir olmalı ve yeni bir izin
	// eklendiğinde ona sessizce erişim kazanmamalı.
	roleSpecs := []struct {
		name        string
		description string
		permissions []string
		users       []uuid.UUID
	}{
		{
			name:        "admin",
			description: "İçerik, LLM profilleri ve puanlama üzerinde tam yetki",
			permissions: []string{
				"content:read", "content:write", "content:publish",
				"llm:read", "llm:write",
				"routing:read", "audit:read",
				"score:read", "score:override",
				"session:read", "session:write",
				// session:read:all, başkasının oturumunu okuma hakkı.
				// session:read'den ayrı: o hak çalışanda da var ve kendi
				// oturumunu okumak anlamına geliyor. İkisini aynı ada
				// bağlamak, her çalışana herkesin transkriptini açardı.
				"session:read:all",
			},
			users: []uuid.UUID{adminUserID},
		},
		{
			name:        "trainee",
			description: "Yayınlanmış içeriği oynar, kendi puanını görür",
			permissions: []string{
				"content:read", "session:read", "session:write", "score:read",
			},
			users: []uuid.UUID{traineeUserID},
		},
	}

	existing, err := roles.ListByScope(ctx, iamModel.ScopeTypeOrganization, demoOrgID)
	if err != nil {
		return fmt.Errorf("roller okunamadı: %w", err)
	}
	byName := make(map[string]*iamModel.Role, len(existing))
	for _, role := range existing {
		byName[role.Name] = role
	}

	for _, spec := range roleSpecs {
		role, ok := byName[spec.name]
		if !ok {
			role = &iamModel.Role{
				ScopeType:   iamModel.ScopeTypeOrganization,
				ScopeID:     demoOrgID,
				Name:        spec.name,
				Description: spec.description,
			}
			if err := roles.Create(ctx, role); err != nil {
				return fmt.Errorf("rol %s: %w", spec.name, err)
			}
		}
		for _, permission := range spec.permissions {
			if err := roles.AddPermission(ctx, role.ID, permission); err != nil {
				return fmt.Errorf("izin %s/%s: %w", spec.name, permission, err)
			}
		}
		for _, userID := range spec.users {
			if err := roles.AssignRoleToUser(ctx, &iamModel.UserRole{
				UserID:         userID,
				RoleID:         role.ID,
				OrganizationID: demoOrgID,
			}); err != nil {
				return fmt.Errorf("rol ataması %s: %w", spec.name, err)
			}
		}
		log.Printf("   rol: %s (%d izin)", spec.name, len(spec.permissions))
	}

	return nil
}

// createIfMissing, varlık zaten varsa oluşturmayı atlar.
//
// Betik tekrar çalıştırılabilir olmalı: doğrulama betiği onu her koşuda
// çağırıyor ve ikinci koşuda çakışma hatasıyla durmamalı.
func createIfMissing(_ context.Context, _ uuid.UUID, create func() error, exists func() error) error {
	if err := exists(); err == nil {
		return nil
	} else if !errors.Is(err, domainErr.ErrNotFound) {
		return err
	}
	if err := create(); err != nil {
		if errors.Is(err, domainErr.ErrAlreadyExists) || errors.Is(err, domainErr.ErrConflict) {
			return nil
		}
		return err
	}
	return nil
}
