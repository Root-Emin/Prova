package config

import (
	"errors"
	"fmt"
	"strings"
)

// IsProduction, katı doğrulamanın uygulanacağı ortamı bildirir.
func (c *Config) IsProduction() bool {
	return c.Environment == EnvProduction
}

// Validate, boot'u durduracak yapılandırma hatalarını toplar.
//
// Buradaki kontrollerin hepsi production'da fail-fast'tir çünkü üçü de sessiz
// başarısızlık üretir: varsayılan JWT secret ile açılan bir sunucu herkesin
// imzalayabildiği token'ları kabul eder, pepper'sız bir kurulum giriş kodu
// digest'lerini gökkuşağı tablosuna açar, e-posta sağlayıcısı "none" iken
// sistem ayakta görünür ama hiç kimse giriş kodu alamaz. Üçü de ancak
// istismar edildiğinde fark edilir; boot anında fark edilmesi çok daha ucuzdur.
//
// Hatalar tek tek değil topluca döndürülür: yapılandırmayı düzelten kişi her
// eksiği ayrı bir yeniden başlatmayla keşfetmemeli.
func (c *Config) Validate() error {
	var problems []error

	if c.IsProduction() {
		if strings.TrimSpace(c.JWT.Secret) == "" || c.JWT.Secret == DefaultJWTSecret {
			problems = append(problems, errors.New(
				"JWT_SECRET boş ya da varsayılan değerde; üretimde bilinen bir imza anahtarı "+
					"tüm kimlik doğrulamasını geçersiz kılar (openssl rand -base64 48)"))
		}
		if strings.TrimSpace(c.Auth.CodePepper) == "" {
			problems = append(problems, errors.New(
				"AUTH_CODE_PEPPER boş; giriş kodu digest'leri 10^6'lık uzayda pepper olmadan "+
					"düz metne eşdeğerdir (openssl rand -base64 32)"))
		}
		if strings.TrimSpace(c.Mongo.URI) == "" || c.Mongo.URI == DefaultMongoURI {
			problems = append(problems, errors.New(
				"MONGO_URI boş ya da localhost varsayılanında; oturum ve puanlama belgeleri "+
					"nesne veritabanında tutulur, yerel varsayılanla üretime çıkmak veri kaybıdır"))
		}
		if strings.EqualFold(strings.TrimSpace(c.Email.Provider), ProviderNone) {
			problems = append(problems, errors.New(
				"EMAIL_PROVIDER=none; giriş parolasızdır, e-posta teslimi olmadan hiç kimse "+
					"sisteme giremez ama sunucu çalışıyor görünür"))
		}
		if strings.TrimSpace(c.Email.FromAddress) == "" {
			problems = append(problems, errors.New("EMAIL_FROM_ADDRESS boş; gönderen adresi olmadan teslim yapılamaz"))
		}
		if len(c.Server.CORSAllowedOrigins) == 0 {
			problems = append(problems, errors.New(
				"CORS_ALLOWED_ORIGINS boş; web arayüzü GraphQL uçlarına erişemez"))
		}
	}

	if c.Auth.CodeLength < 4 || c.Auth.CodeLength > 10 {
		problems = append(problems, fmt.Errorf("AUTH_CODE_LENGTH %d aralık dışında (4-10)", c.Auth.CodeLength))
	}
	if c.GraphQL.MaxDepth < 1 {
		problems = append(problems, errors.New("GRAPHQL_MAX_DEPTH en az 1 olmalı"))
	}
	if c.Token.AccessTTL <= 0 || c.Token.RefreshTTL <= 0 {
		problems = append(problems, errors.New("TOKEN_ACCESS_TTL_SECONDS ve TOKEN_REFRESH_TTL_SECONDS pozitif olmalı"))
	}
	if c.Token.AccessTTL >= c.Token.RefreshTTL {
		problems = append(problems, errors.New(
			"access token ömrü refresh token ömründen kısa olmalı; aksi hâlde rotasyonun bir anlamı kalmaz"))
	}

	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("yapılandırma doğrulaması başarısız:\n  - %s", joinErrors(problems, "\n  - "))
}

// Harden, production'da güvenli olmayan geliştirme kolaylıklarını kapatır.
//
// Bunlar doğrulama hatası değil, sessiz düzeltmedir: playground'un açık
// kalması yapılandırmayı yazan kişinin hatası olmayabilir, ama açık kalması
// hâlinde şema tüm dünyaya okunur.
func (c *Config) Harden() {
	if !c.IsProduction() {
		return
	}
	c.GraphQL.PlaygroundEnabled = false
	c.GraphQL.IntrospectionEnabled = false
}

func joinErrors(errs []error, sep string) string {
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		parts = append(parts, err.Error())
	}
	return strings.Join(parts, sep)
}
