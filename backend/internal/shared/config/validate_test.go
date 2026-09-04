package config

import (
	"strings"
	"testing"
	"time"
)

// productionConfig, doğrulamayı geçen asgari üretim yapılandırması.
func productionConfig() *Config {
	return &Config{
		Environment: EnvProduction,
		Server:      ServerConfig{CORSAllowedOrigins: []string{"https://app.example.com"}},
		Mongo:       MongoConfig{URI: "mongodb://mongo.internal:27017"},
		JWT:         JWTConfig{Secret: "uzun-ve-rastgele-bir-imza-anahtari"},
		Auth:        AuthConfig{CodeLength: 6, CodePepper: "rastgele-pepper"},
		Email:       EmailConfig{Provider: ProviderResend, FromAddress: "noreply@mail.example.com"},
		GraphQL:     GraphQLConfig{MaxDepth: 12},
		Token:       TokenConfig{AccessTTL: 15 * time.Minute, RefreshTTL: 720 * time.Hour},
	}
}

func TestValidate_AcceptsCompleteProductionConfig(t *testing.T) {
	if err := productionConfig().Validate(); err != nil {
		t.Fatalf("eksiksiz üretim yapılandırması reddedildi: %v", err)
	}
}

// Üç sır da sessiz başarısızlık üretir: sunucu ayağa kalkar, sorun ancak
// istismar edildiğinde ortaya çıkar. Doğrulamanın tek işi bunu boot anına
// çekmektir.
func TestValidate_RejectsDefaultSecretsInProduction(t *testing.T) {
	cases := map[string]func(*Config){
		"varsayılan JWT secret":   func(c *Config) { c.JWT.Secret = DefaultJWTSecret },
		"boş JWT secret":          func(c *Config) { c.JWT.Secret = "" },
		"boş pepper":              func(c *Config) { c.Auth.CodePepper = "" },
		"varsayılan Mongo URI":    func(c *Config) { c.Mongo.URI = DefaultMongoURI },
		"e-posta sağlayıcısı yok": func(c *Config) { c.Email.Provider = ProviderNone },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := productionConfig()
			mutate(cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatalf("%s ile boot durmalıydı", name)
			}
		})
	}
}

// Aynı anda birden fazla eksik varsa hepsi tek seferde bildirilir; aksi hâlde
// yapılandırmayı düzelten kişi her eksiği ayrı bir yeniden başlatmayla keşfeder.
func TestValidate_ReportsEveryProblemAtOnce(t *testing.T) {
	cfg := productionConfig()
	cfg.JWT.Secret = DefaultJWTSecret
	cfg.Auth.CodePepper = ""
	cfg.Mongo.URI = ""

	err := cfg.Validate()
	if err == nil {
		t.Fatal("hata bekleniyordu")
	}
	for _, want := range []string{"JWT_SECRET", "AUTH_CODE_PEPPER", "MONGO_URI"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("hata metni %q içermeliydi:\n%s", want, err)
		}
	}
}

// Geliştirmede aynı eksikler engel değildir: yerel kurulumun sır üretmeden
// çalışabilmesi gerekir.
func TestValidate_LenientOutsideProduction(t *testing.T) {
	cfg := Load()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("geliştirme varsayılanları reddedildi: %v", err)
	}
}

// Playground ve introspection yapılandırma hatası değil, sessiz düzeltmedir:
// açık kalmaları hâlinde şema tüm dünyaya okunur.
func TestHarden_ClosesIntrospectionAndPlaygroundInProduction(t *testing.T) {
	cfg := productionConfig()
	cfg.GraphQL.PlaygroundEnabled = true
	cfg.GraphQL.IntrospectionEnabled = true

	cfg.Harden()

	if cfg.GraphQL.PlaygroundEnabled || cfg.GraphQL.IntrospectionEnabled {
		t.Fatal("üretimde playground ve introspection kapalı olmalı")
	}
}

func TestHarden_LeavesDevelopmentAlone(t *testing.T) {
	cfg := Load()
	cfg.GraphQL.PlaygroundEnabled = true
	cfg.Harden()
	if !cfg.GraphQL.PlaygroundEnabled {
		t.Fatal("geliştirmede playground kapatılmamalı")
	}
}

// Access token'ın refresh'ten uzun yaşaması rotasyonu anlamsız kılar.
func TestValidate_RejectsAccessTokenOutlivingRefresh(t *testing.T) {
	cfg := productionConfig()
	cfg.Token.AccessTTL = 48 * time.Hour
	cfg.Token.RefreshTTL = 24 * time.Hour

	if err := cfg.Validate(); err == nil {
		t.Fatal("access token refresh'ten uzun olamaz")
	}
}
