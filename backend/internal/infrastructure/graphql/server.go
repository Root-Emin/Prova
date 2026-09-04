package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/coder/websocket"
	gqlgraph "github.com/masterfabric-go/masterfabric/graph"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/service"
	"github.com/masterfabric-go/masterfabric/internal/shared/authctx"
	"github.com/masterfabric-go/masterfabric/internal/shared/config"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// ServerConfig, GraphQL sunucusunun kurulum parametreleri.
type ServerConfig struct {
	Resolver *gqlgraph.Resolver
	Auth     service.AuthService
	RBAC     service.RBACService
	// Denylist, iptal edilmiş token kimliklerini bilir. Nil ise iptal
	// kontrolü yapılmaz (yalnızca veritabanısız geliştirme kurulumunda).
	Denylist TokenDenylist
	GraphQL  config.GraphQLConfig
	// AllowedOrigins, WebSocket el sıkışmasında kabul edilen kaynaklar.
	AllowedOrigins []string
	Log            *slog.Logger
}

// TokenDenylist, iptal edilmiş access token kimliklerini sorgular.
//
// İmzalı bir token'ı iptal etmenin başka yolu yoktur: cihaz iptal edildiğinde
// o cihazın token'ı hâlâ geçerli imzaya sahiptir ve yalnızca kimliğinden
// tanınabilir.
type TokenDenylist interface {
	IsRevoked(ctx context.Context, claims *service.TokenClaims) (bool, error)
}

// NewServer, /graphql yolunda çalışacak handler'ı kurar.
func NewServer(cfg ServerConfig) http.Handler {
	execCfg := gqlgraph.Config{Resolvers: cfg.Resolver}
	directives := &Directives{RBAC: cfg.RBAC, Log: cfg.Log}
	execCfg.Directives.Auth = directives.Auth
	execCfg.Directives.Permission = directives.Permission

	srv := handler.New(gqlgraph.NewExecutableSchema(execCfg))

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 15 * time.Second,
		// connectionParams içindeki token burada doğrulanır. WebSocket
		// el sıkışması Authorization başlığı taşıyamaz, bu yüzden ilk
		// mesajdaki payload tek kimlik kaynağıdır; doğrulanmadan bırakılırsa
		// abonelikler kimliksiz açılır.
		InitFunc:    websocketInit(cfg),
		InitTimeout: 10 * time.Second,
		Implementation: transport.CoderWebsocketImplementation{
			AcceptOptions: websocket.AcceptOptions{
				OriginPatterns: originPatterns(cfg.AllowedOrigins),
			},
		},
	})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](256))
	srv.Use(extension.AutomaticPersistedQuery{Cache: lru.New[string](128)})

	if cfg.GraphQL.IntrospectionEnabled {
		srv.Use(extension.Introspection{})
	}
	if cfg.GraphQL.MaxComplexity > 0 {
		srv.Use(extension.FixedComplexityLimit(cfg.GraphQL.MaxComplexity))
	}
	if cfg.GraphQL.MaxDepth > 0 {
		srv.Use(DepthLimit{Max: cfg.GraphQL.MaxDepth})
	}

	srv.SetErrorPresenter(errorPresenter(cfg.Log))
	srv.SetRecoverFunc(recoverFunc(cfg.Log))

	// Kimlik, GraphQL'e girmeden önce çözülür: token varsa çağıran bağlama
	// yerleştirilir, yoksa istek kimliksiz devam eder ve @auth directive'i
	// alan bazında karar verir.
	return withClientIP(batchGuard(cfg.GraphQL.MaxBatch)(authenticate(cfg)(srv)))
}

// NewPlayground, geliştirmede tarayıcıdan denemek için GraphiQL sunar.
func NewPlayground() http.Handler {
	return playground.Handler("Prova GraphQL", "/graphql")
}

// withClientIP, isteğin kaynak adresini bağlama koyar.
//
// Resolver'lar HTTP isteğini görmez ama hız sınırlaması ve denetim kaydı
// adrese ihtiyaç duyar. X-Forwarded-For bilerek okunmuyor: ters vekil
// olmadan bu başlık istemci tarafından uydurulabilir ve adrese bağlı her
// limit anlamsızlaşır.
func withClientIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		next.ServeHTTP(w, r.WithContext(authctx.WithClientIP(r.Context(), host)))
	})
}

// batchGuard, tek istekte gönderilen operasyon sayısını sınırlar.
//
// gqlgen dizi biçimli batch isteklerini zaten kabul etmiyor; bu kapı ikinci
// savunma hattı ve açık bir reddetme mesajı. Sınırsız batch, karmaşıklık ve
// derinlik limitlerinin ikisini birden istek başına değil operasyon başına
// hâle getirerek etkisiz kılardı.
func batchGuard(maxBatch int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if maxBatch <= 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}
			body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBody))
			if err != nil {
				http.Error(w, `{"errors":[{"message":"istek gövdesi okunamadı"}]}`, http.StatusBadRequest)
				return
			}
			_ = r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(body))

			if count := batchSize(body); count > maxBatch {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = fmt.Fprintf(w,
					`{"errors":[{"message":"tek istekte en fazla %d operasyon gönderilebilir, %d gönderildi","extensions":{"code":"BAD_REQUEST"}}],"data":null}`,
					maxBatch, count)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// maxRequestBody, batch kontrolü için okunacak gövdenin üst sınırı.
const maxRequestBody = 4 << 20

// batchSize, gövdedeki operasyon sayısını döndürür. Dizi değilse 1.
func batchSize(body []byte) int {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return 1
	}
	var operations []json.RawMessage
	if err := json.Unmarshal(trimmed, &operations); err != nil {
		return 1
	}
	return len(operations)
}

// authenticate, Authorization başlığındaki token'ı çözer.
func authenticate(cfg ServerConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := authctx.BearerToken(r.Header.Get("Authorization"))
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}
			viewer, err := resolveViewer(r.Context(), cfg, token)
			if err != nil {
				// Geçersiz token kimliksiz istekten farklıdır: istemci
				// yenilemesi gerektiğini bilmeli, aksi hâlde sessizce
				// yetkisiz kalır.
				writeAuthError(w, err.Error())
				return
			}
			next.ServeHTTP(w, r.WithContext(authctx.WithViewer(r.Context(), viewer)))
		})
	}
}

// resolveViewer, token'ı doğrular ve iptal listesini kontrol eder.
func resolveViewer(ctx context.Context, cfg ServerConfig, token string) (*authctx.Viewer, error) {
	claims, err := cfg.Auth.ValidateToken(ctx, token)
	if err != nil {
		return nil, errors.New("geçersiz veya süresi dolmuş token")
	}
	if cfg.Denylist != nil {
		revoked, err := cfg.Denylist.IsRevoked(ctx, claims)
		if err != nil {
			cfg.Log.ErrorContext(ctx, "iptal listesi okunamadı", "error", err)
		} else if revoked {
			return nil, errors.New("token iptal edilmiş")
		}
	}
	return authctx.FromClaims(claims), nil
}

// websocketInit, abonelik bağlantısında connectionParams'taki token'ı doğrular.
func websocketInit(cfg ServerConfig) func(context.Context, transport.InitPayload) (context.Context, *transport.InitPayload, error) {
	return func(ctx context.Context, payload transport.InitPayload) (context.Context, *transport.InitPayload, error) {
		raw := payload.Authorization()
		token := authctx.BearerToken(raw)
		if token == "" {
			// Bazı istemciler şemayı "Bearer" öneki olmadan gönderiyor.
			token = raw
		}
		if token == "" {
			return nil, nil, errors.New("connectionParams içinde token yok")
		}
		viewer, err := resolveViewer(ctx, cfg, token)
		if err != nil {
			return nil, nil, err
		}
		return authctx.WithViewer(ctx, viewer), &payload, nil
	}
}

// originPatterns, WebSocket el sıkışmasında kabul edilecek kaynaklar.
//
// Boş bırakılırsa coder/websocket yalnızca aynı kaynağa izin verir; bu,
// yapılandırılmamış bir kurulumda güvenli olan taraftır.
func originPatterns(allowed []string) []string {
	if len(allowed) == 0 {
		return nil
	}
	patterns := make([]string, 0, len(allowed))
	for _, origin := range allowed {
		patterns = append(patterns, stripScheme(origin))
	}
	return patterns
}

func stripScheme(origin string) string {
	for _, prefix := range []string{"https://", "http://"} {
		if len(origin) > len(prefix) && origin[:len(prefix)] == prefix {
			return origin[len(prefix):]
		}
	}
	return origin
}

// errorPresenter, domain hatalarını istemcinin davranabileceği kodlara çevirir.
func errorPresenter(log *slog.Logger) graphql.ErrorPresenterFunc {
	return func(ctx context.Context, err error) *gqlerror.Error {
		gqlErr := graphql.DefaultErrorPresenter(ctx, err)
		if gqlErr.Extensions == nil {
			gqlErr.Extensions = map[string]any{}
		}
		if _, ok := gqlErr.Extensions["code"]; ok {
			return gqlErr
		}

		var domain *domainErr.DomainError
		if errors.As(err, &domain) {
			gqlErr.Extensions["code"] = domainErrorCode(domain)
			gqlErr.Message = domain.Message
			// Sarmalanan neden istemciye gitmez: içinde tablo adı, sorgu
			// metni ve bağlantı dizesi olabilir.
			if domain.Err != nil {
				log.ErrorContext(ctx, "graphql resolver hatası",
					"kind", domain.Kind.Error(), "cause", domain.Err)
			}
			return gqlErr
		}

		gqlErr.Extensions["code"] = "INTERNAL"
		return gqlErr
	}
}

func domainErrorCode(err *domainErr.DomainError) string {
	switch {
	case errors.Is(err, domainErr.ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, domainErr.ErrUnauthorized):
		return CodeUnauthenticated
	case errors.Is(err, domainErr.ErrForbidden):
		return CodeForbidden
	case errors.Is(err, domainErr.ErrValidation), errors.Is(err, domainErr.ErrBadRequest):
		return "BAD_REQUEST"
	case errors.Is(err, domainErr.ErrConflict), errors.Is(err, domainErr.ErrAlreadyExists):
		return "CONFLICT"
	case errors.Is(err, domainErr.ErrRateLimited):
		return "RATE_LIMITED"
	case errors.Is(err, domainErr.ErrNotImplemented):
		return "NOT_IMPLEMENTED"
	default:
		return "INTERNAL"
	}
}

// recoverFunc, resolver panic'ini isteğe hapseder.
func recoverFunc(log *slog.Logger) graphql.RecoverFunc {
	return func(ctx context.Context, err any) error {
		log.ErrorContext(ctx, "graphql resolver panic", "panic", err)
		return gqlerror.Errorf("beklenmeyen bir hata oluştu")
	}
}

func writeAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"errors":[{"message":"` + message + `","extensions":{"code":"` + CodeUnauthenticated + `"}}],"data":null}`))
}
