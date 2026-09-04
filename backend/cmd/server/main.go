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

	"github.com/masterfabric-go/masterfabric/graph"
	iamAppService "github.com/masterfabric-go/masterfabric/internal/application/iam/service"
	iamUC "github.com/masterfabric-go/masterfabric/internal/application/iam/usecase"
	provaAppService "github.com/masterfabric-go/masterfabric/internal/application/prova/service"
	provaUC "github.com/masterfabric-go/masterfabric/internal/application/prova/usecase"
	auditService "github.com/masterfabric-go/masterfabric/internal/domain/audit/service"
	notify "github.com/masterfabric-go/masterfabric/internal/domain/notification/service"
	provaModel "github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
	provaRepo "github.com/masterfabric-go/masterfabric/internal/domain/prova/repository"
	infraAudit "github.com/masterfabric-go/masterfabric/internal/infrastructure/audit"
	infraAuth "github.com/masterfabric-go/masterfabric/internal/infrastructure/auth"
	"github.com/masterfabric-go/masterfabric/internal/infrastructure/email"
	infraGQL "github.com/masterfabric-go/masterfabric/internal/infrastructure/graphql"
	"github.com/masterfabric-go/masterfabric/internal/infrastructure/http/router"
	"github.com/masterfabric-go/masterfabric/internal/infrastructure/jobs"
	"github.com/masterfabric-go/masterfabric/internal/infrastructure/llm"
	infraMongo "github.com/masterfabric-go/masterfabric/internal/infrastructure/mongo"
	provaMongoRepo "github.com/masterfabric-go/masterfabric/internal/infrastructure/mongo/prova"
	pgAudit "github.com/masterfabric-go/masterfabric/internal/infrastructure/postgres/audit"
	pgIam "github.com/masterfabric-go/masterfabric/internal/infrastructure/postgres/iam"
	pgTenant "github.com/masterfabric-go/masterfabric/internal/infrastructure/postgres/tenant"
	"github.com/masterfabric-go/masterfabric/internal/infrastructure/realtime"
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
	// stopBackgroundJobs, zamanlanmış işleri kapanışta durdurur.
	var stopBackgroundJobs func()

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

	denylist := infraAuth.NewDenylist(redisClient)
	magicLinkRepo := pgIam.NewMagicLinkRepo(db)
	deviceChallengeRepo := pgIam.NewDeviceChallengeRepo(db)
	deviceSignatures := infraAuth.NewDeviceSignatureService()
	magicLinkTokens := infraAuth.NewMagicLinkService()
	magicLinkIssuer := iamUC.NewMagicLinkIssuer(
		magicLinkRepo, userRepo, magicLinkTokens, cfg.Token.MagicLinkTTL, log)

	// --- Use case'ler ---
	requestCodeUC := iamUC.NewRequestLoginCodeUseCase(iamUC.RequestDeps{
		Users:      userRepo,
		Codes:      loginCodeRepo,
		CodeSvc:    loginCodeService,
		Sender:     emailSender,
		Limiter:    limiter,
		Audit:      auditRecorder,
		Cfg:        cfg.Auth,
		Log:        log,
		MagicLinks: magicLinkIssuer,
		WebBaseURL: cfg.Token.WebBaseURL,
	})
	refreshRepo := pgIam.NewRefreshTokenRepo(db)
	refreshUC := iamUC.NewRefreshTokenUseCase(iamUC.RefreshDeps{
		Tokens:     refreshRepo,
		Users:      userRepo,
		Devices:    deviceRepo,
		Minter:     jwtService,
		Hasher:     magicLinkTokens,
		Denylist:   denylist,
		Audit:      auditRecorder,
		AccessTTL:  cfg.Token.AccessTTL,
		RefreshTTL: cfg.Token.RefreshTTL,
		Log:        log,
	})

	verifyCodeUC := iamUC.NewVerifyLoginCodeUseCase(iamUC.VerifyDeps{
		Users:       userRepo,
		Codes:       loginCodeRepo,
		Devices:     deviceRepo,
		Challenges:  deviceChallengeRepo,
		Signatures:  deviceSignatures,
		CodeSvc:     loginCodeService,
		Auth:        jwtService,
		Memberships: memberships,
		Sender:      emailSender,
		Limiter:     limiter,
		EventBus:    eventBus,
		Audit:       auditRecorder,
		Refresh:     refreshUC,
		Minter:      jwtService,
		Cfg:         cfg.Auth,
		JWTCfg:      cfg.JWT,
		TokenCfg:    cfg.Token,
		Log:         log,
	})

	verifyMagicLinkUC := iamUC.NewVerifyMagicLinkUseCase(
		magicLinkRepo, userRepo, magicLinkTokens, verifyCodeUC,
		auditRecorder, cfg.Auth.SelfSignup, log)
	orgLookup := iamAppService.NewOrgLookup(orgUserRepo)
	deviceChallengeUC := iamUC.NewRequestDeviceChallengeUseCase(
		deviceChallengeRepo, deviceSignatures, auditRecorder, cfg.Token.DeviceChallengeTTL, log)
	manageDevicesUC := iamUC.NewManageDevicesUseCase(deviceRepo, denylist, auditRecorder, log)
	logoutUC := iamUC.NewLogoutUseCase(denylist, auditRecorder, cfg.Token.AccessTTL, log)

	// --- Prova: nesne veritabanı, LLM hattı ve oturumlar ---
	//
	// MongoDB yoksa Prova akışları hiç bağlanmaz. Yarım bağlanmış bir oturum
	// akışı, kullanıcıya oynanabilir görünüp ilk konuşma sırasında çökerdi.
	var (
		sessionUC      *provaUC.SessionUseCase
		routingStatsUC *provaUC.RoutingStatsUseCase
		characterUC    *provaUC.ContentUseCase[provaModel.Character, *provaModel.Character]
		scenarioUC     *provaUC.ContentUseCase[provaModel.Scenario, *provaModel.Scenario]
		rubricUC       *provaUC.ContentUseCase[provaModel.Rubric, *provaModel.Rubric]
		llmProfileUC   *provaUC.LLMProfileUseCase
		gateway        *provaAppService.Gateway
		broker         *realtime.SessionBroker
		sessionRepo    *provaMongoRepo.SessionRepo
		scoreRepo      *provaMongoRepo.ScoreRepo
		charRepo       *provaMongoRepo.CharacterRepo
		scenRepo       *provaMongoRepo.ScenarioRepo
		rubRepo        *provaMongoRepo.RubricRepo
		profileRepo    *provaMongoRepo.LLMProfileRepo
		routingRepo    *provaMongoRepo.RoutingRepo
	)
	if mongoDB != nil {
		charRepo = provaMongoRepo.NewCharacterRepo(mongoDB.Database)
		scenRepo = provaMongoRepo.NewScenarioRepo(mongoDB.Database)
		rubRepo = provaMongoRepo.NewRubricRepo(mongoDB.Database)
		profileRepo = provaMongoRepo.NewLLMProfileRepo(mongoDB.Database)
		sessionRepo = provaMongoRepo.NewSessionRepo(mongoDB.Database)
		scoreRepo = provaMongoRepo.NewScoreRepo(mongoDB.Database)
		routingRepo = provaMongoRepo.NewRoutingRepo(mongoDB.Database)

		broker = realtime.NewSessionBroker(log)
		gateway = provaAppService.NewGateway(
			profileRepo,
			routingRepo,
			llm.NewClient(cfg.LLM.RequestTimeout),
			provaAppService.NewRuleRouter(provaAppService.RouterConfig{
				LongInputThreshold: cfg.LLM.LongInputThreshold,
				CircuitThreshold:   cfg.LLM.CircuitThreshold,
				CircuitCooldown:    cfg.LLM.CircuitCooldown,
			}),
			// PII maskeleyici LLM hattına burada takılıyor. Maskeleme
			// YALNIZCA giden kopyada: transkript orijinal metni tutar,
			// çünkü "çalışan kişisel veri ifşa etti mi" sorusu ancak
			// orijinal metinle cevaplanabilir.
			llm.NewMasker(),
			llm.NewEnvKeys(),
			log,
		)
		routingStatsUC = provaUC.NewRoutingStatsUseCase(routingRepo, profileRepo)
		characterUC = provaUC.NewContentUseCase[provaModel.Character](charRepo, auditRecorder, "character")
		scenarioUC = provaUC.NewContentUseCase[provaModel.Scenario](scenRepo, auditRecorder, "scenario")
		rubricUC = provaUC.NewContentUseCase[provaModel.Rubric](rubRepo, auditRecorder, "rubric")
		llmProfileUC = provaUC.NewLLMProfileUseCase(profileRepo, gateway, auditRecorder)
		sessionUC = provaUC.NewSessionUseCase(provaUC.SessionDeps{
			Devices:    deviceRepo,
			Sessions:   sessionRepo,
			Scores:     scoreRepo,
			Scenarios:  scenRepo,
			Characters: charRepo,
			Rubrics:    rubRepo,
			Gateway:    gateway,
			Events:     broker,
			Audit:      auditRecorder,
			Log:        log,
		})
	} else {
		log.Warn("nesne veritabanı yok; oturum ve içerik akışları devre dışı")
	}

	// --- Hesap yaşam döngüsü ---
	//
	// Dışa aktarma ve kimliksizleştirme, nesne veritabanına port üzerinden
	// bağlanıyor: kimlik domain'i eğitim içeriğine bağımlı olmamalı.
	accountDeps := iamUC.AccountDeps{
		Users:       userRepo,
		Devices:     deviceRepo,
		Codes:       loginCodeRepo,
		MagicLinks:  magicLinkRepo,
		Refresh:     refreshRepo,
		Audit:       auditRecorder,
		AuditRepo:   auditRepo,
		GracePeriod: cfg.Lifecycle.DeletionGracePeriod,
		Log:         log,
	}
	if sessionRepo != nil {
		accountDeps.Sessions = sessionRepo
		accountDeps.SessionExport = provaUC.NewSessionExporter(sessionRepo, scoreRepo, orgLookup)
	}
	accountUC := iamUC.NewAccountUseCase(accountDeps)

	if cfg.Lifecycle.PurgeEnabled {
		purgeJob := jobs.NewPurgeJob(accountUC, cfg.Lifecycle.PurgeInterval, log)
		jobCtx, stopJobs := context.WithCancel(context.Background())
		go purgeJob.Start(jobCtx)
		stopBackgroundJobs = stopJobs
		log.Info("kalıcı silme işi başlatıldı",
			"interval", cfg.Lifecycle.PurgeInterval,
			"grace_period", cfg.Lifecycle.DeletionGracePeriod)
	}

	// --- GraphQL ---
	resolver := &graph.Resolver{
		Log:                log,
		RequestLoginCodeUC: requestCodeUC,
		VerifyLoginCodeUC:  verifyCodeUC,
		ManageDevicesUC:    manageDevicesUC,
		LogoutUC:           logoutUC,
		VerifyMagicLinkUC:  verifyMagicLinkUC,
		DeviceChallengeUC:  deviceChallengeUC,
		AccountUC:          accountUC,
		RefreshTokenUC:     refreshUC,
		Users:              userRepo,
		RBAC:               rbacService,
		AuditRepo:          auditRepo,

		SessionUC:      sessionUC,
		SessionRepo:    orNilSession(sessionRepo),
		ScoreRepo:      orNilScore(scoreRepo),
		CharacterRepo:  orNilCharacter(charRepo),
		ScenarioRepo:   orNilScenario(scenRepo),
		RubricRepo:     orNilRubric(rubRepo),
		LLMProfileRepo: orNilProfile(profileRepo),
		RoutingRepo:    orNilRouting(routingRepo),
		RoutingStatsUC: routingStatsUC,
		CharacterUC:    characterUC,
		ScenarioUC:     scenarioUC,
		RubricUC:       rubricUC,
		LLMProfileUC:   llmProfileUC,
		Broker:         broker,
	}
	deps.GraphQLHandler = infraGQL.NewServer(infraGQL.ServerConfig{
		Resolver:       resolver,
		Auth:           jwtService,
		RBAC:           rbacService,
		Denylist:       denylist,
		GraphQL:        cfg.GraphQL,
		AllowedOrigins: cfg.Server.CORSAllowedOrigins,
		Log:            log,
	})
	if cfg.GraphQL.PlaygroundEnabled {
		deps.PlaygroundHandler = infraGQL.NewPlayground()
	}

	cleanup := func() {
		if stopBackgroundJobs != nil {
			stopBackgroundJobs()
		}
		// Denetim yazımları eşzamansızdır; süreç ölmeden önce beklenir, yoksa
		// son işlemin kaydı kaybolur.
		auditRecorder.Wait()
	}
	return deps, cleanup
}

var _ auditService.Recorder = (*infraAudit.Recorder)(nil)

// Aşağıdaki yardımcılar tipli nil'i arayüz nil'ine çevirir.
//
// Go'da nil bir *SessionRepo'yu bir arayüz alanına atamak, arayüzü nil
// YAPMAZ: içinde tip bilgisi taşıyan, nil olmayan bir arayüz üretir. Resolver
// bunu "depo var" diye okuyup nil pointer üzerinde metot çağırırdı.
func orNilSession(r *provaMongoRepo.SessionRepo) provaRepo.SessionRepository {
	if r == nil {
		return nil
	}
	return r
}

func orNilScore(r *provaMongoRepo.ScoreRepo) provaRepo.ScoreRepository {
	if r == nil {
		return nil
	}
	return r
}

func orNilCharacter(r *provaMongoRepo.CharacterRepo) provaRepo.CharacterRepository {
	if r == nil {
		return nil
	}
	return r
}

func orNilScenario(r *provaMongoRepo.ScenarioRepo) provaRepo.ScenarioRepository {
	if r == nil {
		return nil
	}
	return r
}

func orNilRubric(r *provaMongoRepo.RubricRepo) provaRepo.RubricRepository {
	if r == nil {
		return nil
	}
	return r
}

func orNilProfile(r *provaMongoRepo.LLMProfileRepo) provaRepo.LLMProfileRepository {
	if r == nil {
		return nil
	}
	return r
}

func orNilRouting(r *provaMongoRepo.RoutingRepo) provaRepo.RoutingRepository {
	if r == nil {
		return nil
	}
	return r
}
