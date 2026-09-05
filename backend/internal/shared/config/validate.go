package config

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// minProductionGracePeriod, üretimde kabul edilen en kısa geri alma penceresi.
//
// Geliştirmede sıfır serbest — silme zincirinin uçtan uca test edilebilmesi
// buna bağlı. Üretimde sıfır, yanlışlıkla "sil" diyen kullanıcının verisini
// aynı dakikada yok etmek demek.
const minProductionGracePeriod = 24 * time.Hour

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
		if strings.TrimSpace(c.EmailVerification.Pepper) == "" {
			problems = append(problems, errors.New(
				"EMAIL_VERIFICATION_OTP_PEPPER boş; e-posta doğrulama digest'leri pepper olmadan güvenli değildir (openssl rand -base64 32)"))
		}
		if strings.TrimSpace(c.Mongo.URI) == "" || c.Mongo.URI == DefaultMongoURI {
			problems = append(problems, errors.New(
				"MONGO_URI boş ya da localhost varsayılanında; oturum ve puanlama belgeleri "+
					"nesne veritabanında tutulur, yerel varsayılanla üretime çıkmak veri kaybıdır"))
		}
		provider := strings.ToLower(strings.TrimSpace(c.Email.Provider))
		if provider == ProviderResend && strings.TrimSpace(c.Email.Resend.APIKey) == "" {
			problems = append(problems, errors.New("RESEND_API_KEY boş; Resend teslimi yapılamaz"))
		}
		if strings.EqualFold(strings.TrimSpace(c.Email.Provider), ProviderNone) {
			problems = append(problems, errors.New(
				"EMAIL_PROVIDER=none; giriş parolasızdır, e-posta teslimi olmadan hiç kimse "+
					"sisteme giremez ama sunucu çalışıyor görünür"))
		}
		if strings.TrimSpace(c.Email.FromAddress) == "" {
			problems = append(problems, errors.New("RESEND_FROM_EMAIL boş; gönderen adresi olmadan teslim yapılamaz"))
		}
		if c.Lifecycle.DeletionGracePeriod < minProductionGracePeriod {
			problems = append(problems, fmt.Errorf(
				"LIFECYCLE_DELETION_GRACE_SECONDS üretimde en az %s olmalı; daha kısa bir pencere, "+
					"yanlışlıkla silme talebi veren kullanıcıya geri alma şansı bırakmaz",
				minProductionGracePeriod))
		}
		if len(c.Server.CORSAllowedOrigins) == 0 {
			problems = append(problems, errors.New(
				"CORS_ALLOWED_ORIGINS boş; web arayüzü GraphQL uçlarına erişemez"))
		}
	}

	if c.Auth.CodeLength != 6 {
		problems = append(problems, fmt.Errorf("AUTH_CODE_LENGTH %d geçersiz; giriş kodu tam 6 haneli olmalı", c.Auth.CodeLength))
	}
	if c.EmailVerification.OTPTTL != 5*time.Minute {
		problems = append(problems, fmt.Errorf("EMAIL_VERIFICATION_OTP_TTL_SECONDS %d geçersiz; doğrulama kodu tam 300 saniye geçerli olmalı", int(c.EmailVerification.OTPTTL.Seconds())))
	}
	if c.EmailVerification.MaxAttempts != 5 {
		problems = append(problems, fmt.Errorf("EMAIL_VERIFICATION_MAX_ATTEMPTS %d geçersiz; doğrulama kodu tam 5 denemeyle sınırlı olmalı", c.EmailVerification.MaxAttempts))
	}
	if c.EmailVerification.ResendCooldown != time.Minute {
		problems = append(problems, fmt.Errorf("EMAIL_VERIFICATION_RESEND_COOLDOWN_SECONDS %d geçersiz; yeniden gönderim aralığı tam 60 saniye olmalı", int(c.EmailVerification.ResendCooldown.Seconds())))
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
