// Command server, Prova backend'ini ayağa kaldırır.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	iamAppService "github.com/masterfabric-go/masterfabric/internal/application/iam/service"
	iamUC "github.com/masterfabric-go/masterfabric/internal/application/iam/usecase"
	auditService "github.com/masterfabric-go/masterfabric/internal/domain/audit/service"
	notify "github.com/masterfabric-go/masterfabric/internal/domain/notification/service"
	infraAudit "github.com/masterfabric-go/masterfabric/internal/infrastructure/audit"
	infraAuth "github.com/masterfabric-go/masterfabric/internal/infrastructure/auth"
	"github.com/masterfabric-go/masterfabric/internal/infrastructure/email"
	"github.com/masterfabric-go/masterfabric/internal/infrastructure/http/router"
	infraMongo "github.com/masterfabric-go/masterfabric/internal/infrastructure/mongo"
	pgAudit "github.com/masterfabric-go/masterfabric/internal/infrastructure/postgres/audit"
	pgIam "github.com/masterfabric-go/masterfabric/internal/infrastructure/postgres/iam"
	pgTenant "github.com/masterfabric-go/masterfabric/internal/infrastructure/postgres/tenant"
	"github.com/masterfabric-go/masterfabric/internal/shared/cache"
	"github.com/masterfabric-go/masterfabric/internal/shared/config"
	"github.com/masterfabric-go/masterfabric/internal/shared/database"
	"github.com/masterfabric-go/masterfabric/internal/shared/events"
	"github.com/masterfabric-go/masterfabric/internal/shared/logger"
	"github.com/masterfabric-go/masterfabric/internal/shared/ratelimit"
	"github.com/masterfabric-go/masterfabric/internal/shared/telemetry"
	"github.com/masterfabric-go/masterfabric/internal/shared/version"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()

	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	slog.SetDefault(log)

	// Doğrulama, herhangi bir bağlantı açılmadan önce çalışır. Eksik bir sır
	// yüzünden güvensiz başlayan bir sunucu, hatayı ancak istismar edildiğinde
	// gösterir; burada durmak en ucuz keşiftir.
	if err := cfg.Validate(); err != nil {
		return err
	}
	cfg.Harden()

	log.Info("starting prova backend",
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
		"environment", cfg.Environment,
	)
	if !cfg.IsProduction() && cfg.JWT.Secret == config.DefaultJWTSecret {
		log.Warn("JWT_SECRET varsayılan değerde; yalnızca geliştirme için kabul edilir")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	otelShutdown, err := telemetry.Setup(ctx, version.ServiceName, version.Version)
	if err != nil {
		log.Warn("opentelemetry setup failed", "error", err)
	} else {
		defer func() { _ = otelShutdown(context.Background()) }()
	}

	db, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		log.Warn("postgres unavailable, running without database", "error", err)
		db = nil
	} else {
		defer db.Close()
		log.Info("connected to postgres")
	}

	mongoDB, err := database.NewMongoClient(ctx, cfg.Mongo)
	if err != nil {
		log.Warn("mongodb unavailable, running without object database", "error", err)
		mongoDB = nil
	} else {
		defer func() { _ = mongoDB.Close(context.Background()) }()
		log.Info("connected to mongodb", "database", cfg.Mongo.Database)

		if err := infraMongo.EnsureIndexes(ctx, mongoDB.Database); err != nil {
			// Index oluşturma burada kozmetik değil: (org_id, lineage_id,
			// version) benzersizliği, iki eşzamanlı yayının aynı sürüm
			// numarasını üretmesini engelleyen tek şeydir.
			log.Error("failed to ensure mongodb indexes", "error", err)
		}
	}

	redisClient, err := cache.NewRedisClient(ctx, cfg.Redis)
	if err != nil {
		log.Warn("redis unavailable, running without cache", "error", err)
		redisClient = nil
	} else {
		defer redisClient.Close()
		log.Info("connected to redis")
	}

	// Olay veri yolu her zaman süreç içidir. Kafka bağımlılığı Prova'nın
	// ihtiyacı olmayan bir dağıtık kurulum getiriyordu; kod repoda duruyor ama
	// bağlanmıyor.
	eventBus := events.NewInProcessBus(log, 256)
	defer func() { _ = eventBus.Close() }()

	emailSender, err := email.New(cfg.Email)
	if err != nil {
		return fmt.Errorf("email sender: %w", err)
	}
	log.Info("email sender initialized", "provider", emailSender.Name())

	deps, cleanup := buildDependencies(log, cfg, db, mongoDB, redisClient, eventBus, emailSender)
	defer cleanup()

	r := router.New(deps)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", addr)
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	case sig := <-shutdown:
		log.Info("shutdown signal received", "signal", sig)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			_ = srv.Close()
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
		log.Info("server stopped gracefully")
	}

	return nil
}

// buildDependencies, tüm bağımlılık grafiğini kurar ve kapanışta çalışacak
// temizleyiciyi döndürür.
func buildDependencies(
	log *slog.Logger,
	cfg *config.Config,
	db *pgxpool.Pool,
	mongoDB *database.MongoDB,
	redisClient *redis.Client,
	eventBus events.EventBus,
	emailSender notify.Sender,
) (router.Dependencies, func()) {
	noop := func() {}

	deps := router.Dependencies{
		Logger:             log,
		DB:                 db,
		Redis:              redisClient,
		CORSAllowedOrigins: cfg.Server.CORSAllowedOrigins,
		MaxBodyBytes:       cfg.Server.MaxBodyBytes,
	}

	if db == nil {
		log.Warn("veritabanı yok; yalnızca sağlık uçları çalışacak")
		return deps, noop
	}

	// --- Depolar ---
	userRepo := pgIam.NewUserRepo(db)
	roleRepo := pgIam.NewRoleRepo(db)
	loginCodeRepo := pgIam.NewLoginCodeRepo(db)
	deviceRepo := pgIam.NewDeviceRepo(db)
	orgUserRepo := pgIam.NewOrgUserRepo(db)
	orgRepo := pgTenant.NewOrgRepo(db)
	auditRepo := pgAudit.NewAuditRepo(db)

	// --- Servisler ---
	jwtService := infraAuth.NewJWTService(cfg.JWT)
	rbacService := infraAuth.NewRBACService(roleRepo, redisClient)
	auditRecorder := infraAudit.NewRecorder(auditRepo, log)
	memberships := iamAppService.NewMembershipService(orgRepo, orgUserRepo, roleRepo, log)

	loginCodeService, err := infraAuth.NewLoginCodeService(cfg.Auth)
	if err != nil {
		// Pepper olmadan saklanan digest'ler kodun kendisinden bir gökkuşağı
		// tablosu uzaktadır, ve tüm giriş akışı o digest'lere dayanır.
		log.Error("parolasız kimlik doğrulama devre dışı", "error", err)
		return deps, noop
	}

	var limiter ratelimit.Limiter
	if redisClient != nil {
		limiter = ratelimit.NewRedisLimiter(redisClient, "ratelimit:auth")
	} else {
		log.Warn("redis yok; kimlik doğrulama limitleri yalnızca bu örnek için geçerli")
		limiter = ratelimit.NewMemoryLimiter()
	}

	// --- Use case'ler ---
	_ = iamUC.NewRequestLoginCodeUseCase(iamUC.RequestDeps{
		Users:      userRepo,
		Codes:      loginCodeRepo,
		CodeSvc:    loginCodeService,
		Sender:     emailSender,
		Limiter:    limiter,
		Audit:      auditRecorder,
		Cfg:        cfg.Auth,
		Log:        log,
		WebBaseURL: cfg.Token.WebBaseURL,
	})
	_ = iamUC.NewVerifyLoginCodeUseCase(iamUC.VerifyDeps{
		Users:       userRepo,
		Codes:       loginCodeRepo,
		Devices:     deviceRepo,
		CodeSvc:     loginCodeService,
		Auth:        jwtService,
		Memberships: memberships,
		Sender:      emailSender,
		Limiter:     limiter,
		EventBus:    eventBus,
		Audit:       auditRecorder,
		Cfg:         cfg.Auth,
		JWTCfg:      cfg.JWT,
		Log:         log,
	})

	_ = rbacService
	_ = mongoDB

	cleanup := func() {
		// Denetim yazımları eşzamansızdır; süreç ölmeden önce beklenir, yoksa
		// son işlemin kaydı kaybolur.
		auditRecorder.Wait()
	}
	return deps, cleanup
}

var _ auditService.Recorder = (*infraAudit.Recorder)(nil)
