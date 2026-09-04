// Package router, HTTP yüzeyini kurar.
//
// Prova'nın istemci API'si GraphQL'dir. Burada kalan REST uçları yalnızca
// sağlık kontrolü ve metriklerdir: ikisi de kimlik doğrulaması olmadan, sabit
// yolda ve makine tarafından okunacak biçimde çalışmak zorundadır, bu yüzden
// GraphQL'e taşınmaları anlamsız olurdu.
package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/masterfabric-go/masterfabric/internal/infrastructure/http/handler/health"
	"github.com/masterfabric-go/masterfabric/internal/shared/middleware"
)

// Dependencies holds all injected dependencies for the router.
//
// Gateway, API management ve Kafka bağımlılıkları bilerek yoktur. Üçü de
// masterfabric-go'nun API ağ geçidi ürününe aitti; Prova bir ağ geçidi değil,
// bir eğitim ve sertifikasyon ürünü. Dosyaları repoda duruyor, ama hiçbir
// yerden bağlanmıyor: ölü kod, çalışan koda karışan koddan iyidir.
type Dependencies struct {
	Logger *slog.Logger
	DB     *pgxpool.Pool
	Redis  *redis.Client

	CORSAllowedOrigins []string
	MaxBodyBytes       int64

	// GraphQLHandler, tüm istemci API yüzeyi. Nil ise /graphql mount edilmez
	// ve sunucu yalnızca sağlık uçlarıyla ayakta kalır.
	GraphQLHandler http.Handler
	// PlaygroundHandler, geliştirmede tarayıcıdan denemek için. Production'da
	// nil'dir (bkz. config.Harden).
	PlaygroundHandler http.Handler
}

// New creates the root Chi router with all middleware and routes.
func New(deps Dependencies) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware. chi, bir grupta rota tanımlandıktan sonra middleware
	// eklenmesine panic ile karşılık verir; bu yüzden her grupta Use çağrıları
	// istisnasız rota tanımlarından önce gelir.
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging(deps.Logger))
	r.Use(middleware.Recoverer(deps.Logger))
	if deps.MaxBodyBytes > 0 {
		r.Use(middleware.MaxBodyBytes(deps.MaxBodyBytes))
	}
	r.Use(cors.Handler(middleware.CORSOptions(deps.CORSAllowedOrigins)))

	// Sağlık uçları: yük dengeleyici ve konteyner orkestratörü için.
	healthHandler := health.NewHandler(deps.DB, deps.Redis)
	r.Get("/health/live", healthHandler.Liveness)
	r.Get("/health/ready", healthHandler.Readiness)

	r.Handle("/metrics", promhttp.Handler())

	// GraphQL. Kimlik doğrulaması burada middleware olarak zorlanmaz: şema
	// içinde @auth directive'i alan bazında karar verir, çünkü kod isteme ve
	// kod doğrulama mutation'ları token'sız çağrılmak zorundadır.
	if deps.GraphQLHandler != nil {
		r.Handle("/graphql", deps.GraphQLHandler)
		// WebSocket transport aynı yolu kullanır; ayrı bir uç açmak istemciyi
		// iki adres tutmaya zorlardı.
		r.Handle("/graphql/*", deps.GraphQLHandler)
	}
	if deps.PlaygroundHandler != nil {
		r.Handle("/playground", deps.PlaygroundHandler)
	}

	// Standart JSON 404. Eski sürümdeki /* yakalayıcı ve global NotFound
	// handler'ı ağ geçidine aitti ve GraphQL mount'uyla çakışıyordu: /graphql
	// altındaki her yol önce yakalayıcıya düşüyordu.
	r.NotFound(notFound)
	r.MethodNotAllowed(methodNotAllowed)

	return r
}

func notFound(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{"error":"not found","code":404}`))
}

func methodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	_, _ = w.Write([]byte(`{"error":"method not allowed","code":405}`))
}
