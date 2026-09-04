package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration.
type Config struct {
	// Environment, dağıtımın hangi ortamda çalıştığını söyler. Fail-fast
	// doğrulaması yalnızca production'da katıdır; geliştirme ortamında aynı
	// eksikler uyarı olarak geçilir.
	Environment string

	Server    ServerConfig
	Database  DatabaseConfig
	Mongo     MongoConfig
	Redis     RedisConfig
	JWT       JWTConfig
	Auth      AuthConfig
	Email     EmailConfig
	Kafka     KafkaConfig
	WebSocket WebSocketConfig
	Log       LogConfig

	// Prova'ya özgü bölümler.
	GraphQL   GraphQLConfig
	LLM       LLMConfig
	Lifecycle LifecycleConfig
	Token     TokenConfig
}

// WebSocketConfig holds real-time WebSocket settings.
type WebSocketConfig struct {
	Enabled         bool
	MaxConnections  int
	PingIntervalSec int
	ReadBufferSize  int
	WriteBufferSize int
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host               string
	Port               int
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	CORSAllowedOrigins []string
	MaxBodyBytes       int64
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int32
	MinConns int32
}

// DSN returns the PostgreSQL connection string with escaped credentials.
func (d DatabaseConfig) DSN() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(d.User, d.Password),
		Host:   fmt.Sprintf("%s:%d", d.Host, d.Port),
		Path:   "/" + d.DBName,
	}
	u.RawQuery = url.Values{"sslmode": {d.SSLMode}}.Encode()
	return u.String()
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// Addr returns the Redis address string.
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// DefaultJWTSecret is the placeholder shipped for local development. It is a
// known value, so production must never boot with it; Validate refuses to.
const DefaultJWTSecret = "change-me-in-production"

// DefaultMongoURI is the loopback URI used for local development.
const DefaultMongoURI = "mongodb://localhost:27017"

// EnvProduction is the value of APP_ENV that turns on strict validation.
const EnvProduction = "production"

// JWTConfig holds JWT signing settings.
type JWTConfig struct {
	Secret          string
	ExpirationHours int
	Issuer          string
}

// MongoConfig holds the object database (MongoDB) connection settings.
//
// MongoDB stores the versioned training documents (scenario, character, rubric,
// session). Identity lives in PostgreSQL; see DatabaseConfig.
type MongoConfig struct {
	URI            string
	Database       string
	ConnectTimeout time.Duration
	MaxPoolSize    uint64
	MinPoolSize    uint64
}

// AuthConfig holds the passwordless (e-mail + one-time code) authentication settings.
//
// There is no password anywhere in this system: the only credential is a
// short-lived numeric code delivered to a verified mailbox. Every field below
// exists to keep that single credential safe.
type AuthConfig struct {
	// CodeLength is the number of digits in a login code.
	CodeLength int
	// CodePepper is the server-side secret mixed into the stored code digest.
	// Without it a leaked digest column is trivially brute-forced, since the
	// code space is only 10^CodeLength.
	CodePepper string
	// CodeTTL is how long a code stays usable after it is issued.
	CodeTTL time.Duration
	// MaxAttempts is how many wrong guesses a single code tolerates before it
	// is burned.
	MaxAttempts int
	// ResendCooldown is the minimum wait between two code requests for the
	// same address.
	ResendCooldown time.Duration
	// MaxRequestsPerEmail / MaxRequestsPerIP cap code requests inside
	// RateLimitWindow.
	MaxRequestsPerEmail int
	MaxRequestsPerIP    int
	RateLimitWindow     time.Duration
	// MinResponseTime pads every login response to a fixed floor so that the
	// answer for a registered address is indistinguishable from the answer for
	// an unknown one. Account enumeration is a timing problem, not only a
	// wording problem.
	MinResponseTime time.Duration
	// SelfSignup provisions a user on first successful code verification.
	// Turn this off once organisation invitations exist.
	SelfSignup bool
}

// E-posta sağlayıcı adları. Adlar burada durur çünkü hem fabrika hem de
// doğrulama bunlara bakar; fabrika paketine bakmak config'i infrastructure'a
// bağımlı kılardı.
const (
	ProviderResend = "resend"
	ProviderSMTP   = "smtp"
	ProviderNone   = "none"
)

// EmailConfig holds transactional e-mail settings.
//
// Provider selects the delivery adapter. The application never imports a
// provider package directly; it depends on the notification.Sender port only,
// so switching away from Resend is a configuration change plus one new adapter.
type EmailConfig struct {
	Provider    string // resend | smtp | none
	FromAddress string
	FromName    string
	ReplyTo     string
	Timeout     time.Duration
	Resend      ResendConfig
	SMTP        SMTPConfig
}

// ResendConfig holds the Resend-specific credentials.
type ResendConfig struct {
	APIKey  string
	BaseURL string
	// Region is the Resend sending region the domain was registered in. It is
	// not sent with API calls; it is recorded so that the DNS records (the
	// bounce MX in particular) and the deployment stay in sync.
	Region string
}

// KafkaConfig holds Kafka connection and consumer settings.
type KafkaConfig struct {
	Brokers           []string
	GroupID           string
	Enabled           bool
	NumPartitions     int
	ReplicationFactor int
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string // debug, info, warn, error
	Format string // json, text
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Environment: strings.ToLower(envOrDefault("APP_ENV", "development")),
		Server: ServerConfig{
			Host:               envOrDefault("SERVER_HOST", "0.0.0.0"),
			Port:               envOrDefaultInt("SERVER_PORT", 8080),
			ReadTimeout:        time.Duration(envOrDefaultInt("SERVER_READ_TIMEOUT_SECONDS", 15)) * time.Second,
			WriteTimeout:       time.Duration(envOrDefaultInt("SERVER_WRITE_TIMEOUT_SECONDS", 15)) * time.Second,
			IdleTimeout:        time.Duration(envOrDefaultInt("SERVER_IDLE_TIMEOUT_SECONDS", 60)) * time.Second,
			CORSAllowedOrigins: envOrDefaultSlice("CORS_ALLOWED_ORIGINS", nil),
			MaxBodyBytes:       envOrDefaultInt64("MAX_BODY_BYTES", 1<<20),
		},
		Database: DatabaseConfig{
			Host:     envOrDefault("DB_HOST", "localhost"),
			Port:     envOrDefaultInt("DB_PORT", 5432),
			User:     envOrDefault("DB_USER", "masterfabric"),
			Password: envOrDefault("DB_PASSWORD", "masterfabric"),
			DBName:   envOrDefault("DB_NAME", "masterfabric"),
			SSLMode:  envOrDefault("DB_SSLMODE", "disable"),
			MaxConns: envOrDefaultInt32("DB_MAX_CONNS", 25),
			MinConns: envOrDefaultInt32("DB_MIN_CONNS", 5),
		},
		Mongo: MongoConfig{
			URI:            envOrDefault("MONGO_URI", DefaultMongoURI),
			Database:       envOrDefault("MONGO_DATABASE", "prova"),
			ConnectTimeout: time.Duration(envOrDefaultInt("MONGO_CONNECT_TIMEOUT_SECONDS", 10)) * time.Second,
			MaxPoolSize:    uint64(envOrDefaultInt("MONGO_MAX_POOL_SIZE", 50)),
			MinPoolSize:    uint64(envOrDefaultInt("MONGO_MIN_POOL_SIZE", 5)),
		},
		Redis: RedisConfig{
			Host:     envOrDefault("REDIS_HOST", "localhost"),
			Port:     envOrDefaultInt("REDIS_PORT", 6379),
			Password: envOrDefault("REDIS_PASSWORD", ""),
			DB:       envOrDefaultInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:          envOrDefault("JWT_SECRET", DefaultJWTSecret),
			ExpirationHours: envOrDefaultInt("JWT_EXPIRATION_HOURS", 24),
			Issuer:          envOrDefault("JWT_ISSUER", "masterfabric"),
		},
		Auth: AuthConfig{
			CodeLength:          envOrDefaultInt("AUTH_CODE_LENGTH", 6),
			CodePepper:          envOrDefault("AUTH_CODE_PEPPER", ""),
			CodeTTL:             time.Duration(envOrDefaultInt("AUTH_CODE_TTL_SECONDS", 600)) * time.Second,
			MaxAttempts:         envOrDefaultInt("AUTH_CODE_MAX_ATTEMPTS", 5),
			ResendCooldown:      time.Duration(envOrDefaultInt("AUTH_CODE_RESEND_COOLDOWN_SECONDS", 60)) * time.Second,
			MaxRequestsPerEmail: envOrDefaultInt("AUTH_CODE_MAX_REQUESTS_PER_EMAIL", 5),
			MaxRequestsPerIP:    envOrDefaultInt("AUTH_CODE_MAX_REQUESTS_PER_IP", 20),
			RateLimitWindow:     time.Duration(envOrDefaultInt("AUTH_CODE_RATE_LIMIT_WINDOW_SECONDS", 3600)) * time.Second,
			MinResponseTime:     time.Duration(envOrDefaultInt("AUTH_MIN_RESPONSE_TIME_MS", 400)) * time.Millisecond,
			SelfSignup:          envOrDefault("AUTH_SELF_SIGNUP", "true") == "true",
		},
		Email: EmailConfig{
			Provider:    envOrDefault("EMAIL_PROVIDER", "resend"),
			FromAddress: envOrDefault("EMAIL_FROM_ADDRESS", ""),
			FromName:    envOrDefault("EMAIL_FROM_NAME", "Prova"),
			ReplyTo:     envOrDefault("EMAIL_REPLY_TO", ""),
			Timeout:     time.Duration(envOrDefaultInt("EMAIL_TIMEOUT_SECONDS", 10)) * time.Second,
			Resend: ResendConfig{
				APIKey:  envOrDefault("RESEND_API_KEY", ""),
				BaseURL: envOrDefault("RESEND_BASE_URL", "https://api.resend.com"),
				Region:  envOrDefault("RESEND_REGION", "eu-west-1"),
			},
			SMTP: SMTPConfig{
				Host:     envOrDefault("SMTP_HOST", "localhost"),
				Port:     envOrDefaultInt("SMTP_PORT", 1025),
				Username: envOrDefault("SMTP_USERNAME", ""),
				Password: envOrDefault("SMTP_PASSWORD", ""),
				UseTLS:   envOrDefault("SMTP_USE_TLS", "false") == "true",
			},
		},
		Kafka: KafkaConfig{
			Brokers:           envOrDefaultSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			GroupID:           envOrDefault("KAFKA_GROUP_ID", "masterfabric-go"),
			Enabled:           envOrDefault("KAFKA_ENABLED", "false") == "true",
			NumPartitions:     envOrDefaultInt("KAFKA_NUM_PARTITIONS", 3),
			ReplicationFactor: envOrDefaultInt("KAFKA_REPLICATION_FACTOR", 1),
		},
		WebSocket: WebSocketConfig{
			Enabled:         envOrDefault("WS_ENABLED", "true") == "true",
			MaxConnections:  envOrDefaultInt("WS_MAX_CONNECTIONS", 1000),
			PingIntervalSec: envOrDefaultInt("WS_PING_INTERVAL_SECONDS", 30),
			ReadBufferSize:  envOrDefaultInt("WS_READ_BUFFER_SIZE", 1024),
			WriteBufferSize: envOrDefaultInt("WS_WRITE_BUFFER_SIZE", 1024),
		},
		Log: LogConfig{
			Level:  envOrDefault("LOG_LEVEL", "info"),
			Format: envOrDefault("LOG_FORMAT", "json"),
		},
		GraphQL: GraphQLConfig{
			// Varsayılanlar geliştirme içindir. Validate(), production'da
			// playground ve introspection'ı zorla kapatır.
			PlaygroundEnabled:    envOrDefault("GRAPHQL_PLAYGROUND", "true") == "true",
			IntrospectionEnabled: envOrDefault("GRAPHQL_INTROSPECTION", "true") == "true",
			MaxDepth:             envOrDefaultInt("GRAPHQL_MAX_DEPTH", 12),
			MaxComplexity:        envOrDefaultInt("GRAPHQL_MAX_COMPLEXITY", 500),
			MaxBatch:             envOrDefaultInt("GRAPHQL_MAX_BATCH", 5),
		},
		LLM: LLMConfig{
			RequestTimeout:     time.Duration(envOrDefaultInt("LLM_REQUEST_TIMEOUT_SECONDS", 60)) * time.Second,
			CircuitThreshold:   envOrDefaultInt("LLM_CIRCUIT_THRESHOLD", 3),
			CircuitCooldown:    time.Duration(envOrDefaultInt("LLM_CIRCUIT_COOLDOWN_SECONDS", 30)) * time.Second,
			LongInputThreshold: envOrDefaultInt("LLM_LONG_INPUT_THRESHOLD", 4000),
			StoreRawAudio:      envOrDefault("LLM_STORE_RAW_AUDIO", "false") == "true",
		},
		Lifecycle: LifecycleConfig{
			DeletionGracePeriod: time.Duration(envOrDefaultInt("LIFECYCLE_DELETION_GRACE_SECONDS", 30*24*3600)) * time.Second,
			PurgeInterval:       time.Duration(envOrDefaultInt("LIFECYCLE_PURGE_INTERVAL_SECONDS", 3600)) * time.Second,
			PurgeEnabled:        envOrDefault("LIFECYCLE_PURGE_ENABLED", "true") == "true",
		},
		Token: TokenConfig{
			AccessTTL:          time.Duration(envOrDefaultInt("TOKEN_ACCESS_TTL_SECONDS", 900)) * time.Second,
			RefreshTTL:         time.Duration(envOrDefaultInt("TOKEN_REFRESH_TTL_SECONDS", 30*24*3600)) * time.Second,
			MagicLinkTTL:       time.Duration(envOrDefaultInt("TOKEN_MAGIC_LINK_TTL_SECONDS", 900)) * time.Second,
			DeviceChallengeTTL: time.Duration(envOrDefaultInt("TOKEN_DEVICE_CHALLENGE_TTL_SECONDS", 120)) * time.Second,
			MaxFailedAttempts:  envOrDefaultInt("TOKEN_MAX_FAILED_ATTEMPTS", 10),
			LockDuration:       time.Duration(envOrDefaultInt("TOKEN_LOCK_DURATION_SECONDS", 900)) * time.Second,
			WebBaseURL:         envOrDefault("WEB_BASE_URL", "http://localhost:3000"),
		},
	}
}

func envOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func envOrDefaultInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func envOrDefaultInt32(key string, defaultVal int32) int32 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			return int32(n)
		}
	}
	return defaultVal
}

func envOrDefaultInt64(key string, defaultVal int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return defaultVal
}

func envOrDefaultSlice(key string, defaultVal []string) []string {
	if val := os.Getenv(key); val != "" {
		parts := strings.Split(val, ",")
		var result []string
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultVal
}

// GraphQLConfig, tek istemci API yüzeyi olan GraphQL sunucusunun sertleştirme
// ayarlarını taşır.
//
// Derinlik ve karmaşıklık limitleri isteğe bağlı süs değildir: iç içe geçmiş
// bir sorgu tek istekle veritabanını kilitleyebilir, ve introspection açık
// bırakılan bir production şeması saldırgana tüm saldırı yüzeyinin haritasını
// verir.
type GraphQLConfig struct {
	// PlaygroundEnabled, tarayıcıdan denenebilen GraphiQL arayüzünü açar.
	PlaygroundEnabled bool
	// IntrospectionEnabled, __schema sorgularına izin verir.
	IntrospectionEnabled bool
	// MaxDepth, bir sorgunun izin verilen en fazla iç içe geçme derinliği.
	MaxDepth int
	// MaxComplexity, alan sayısı ağırlıklı toplam karmaşıklık tavanı.
	MaxComplexity int
	// MaxBatch, tek HTTP isteğinde gönderilebilecek en fazla operasyon sayısı.
	MaxBatch int
}

// LLMConfig, iki kademeli model hattının çalışma zamanı ayarları.
//
// Model adı ve sağlayıcı burada yoktur; onlar LLM profilleri koleksiyonundan
// okunur. Buradaki değerler yalnızca taşıma katmanının davranışını belirler.
type LLMConfig struct {
	// RequestTimeout, tek bir sağlayıcı çağrısının üst sınırı.
	RequestTimeout time.Duration
	// CircuitThreshold, devre kesicinin açılması için gereken ardışık hata sayısı.
	CircuitThreshold int
	// CircuitCooldown, devre açıldıktan sonra tek deneme yapılana kadar geçen süre.
	CircuitCooldown time.Duration
	// LongInputThreshold, bu karakter sayısını aşan girdiyi güçlü kademeye
	// yönlendirir; hızlı model uzun bağlamda gözle görülür şekilde bozulur.
	LongInputThreshold int
	// StoreRawAudio, ham ses kaydının saklanıp saklanmayacağı. Varsayılan
	// kapalıdır ve öyle kalmalıdır: istemci backend'e yalnızca metin gönderir.
	StoreRawAudio bool
}

// LifecycleConfig, hesap yaşam döngüsünün zaman ayarları.
type LifecycleConfig struct {
	// DeletionGracePeriod, silme talebi ile kalıcı silme arasındaki geri alma
	// penceresi. Test edilebilmesi için env'den ayarlanır.
	DeletionGracePeriod time.Duration
	// PurgeInterval, kalıcı silme işinin çalışma sıklığı.
	PurgeInterval time.Duration
	// PurgeEnabled, zamanlanmış silme işini açar.
	PurgeEnabled bool
}

// TokenConfig, kısa ömürlü access + rotasyonlu refresh token ayarları.
type TokenConfig struct {
	// AccessTTL, access token'ın ömrü. Kısa tutulur çünkü iptal, denylist
	// dışında yalnızca sona ermeyle gerçekleşir.
	AccessTTL time.Duration
	// RefreshTTL, refresh token ailesinin toplam ömrü.
	RefreshTTL time.Duration
	// MagicLinkTTL, e-postadaki tek kullanımlık bağlantının ömrü.
	MagicLinkTTL time.Duration
	// DeviceChallengeTTL, cihaz imza challenge'ının ömrü. Kısadır: challenge
	// yeniden oynatılabilir bir kimlik bilgisine dönüşmemeli.
	DeviceChallengeTTL time.Duration
	// MaxFailedAttempts, hesabın geçici kilitlenmesi için gereken ardışık
	// başarısız giriş sayısı.
	MaxFailedAttempts int
	// LockDuration, kilidin süresi.
	LockDuration time.Duration
	// WebBaseURL, magic link'in işaret ettiği web arayüzünün kökü. Electron
	// deep-link'i bilerek kullanılmaz; bağlantı tarayıcıda açılır.
	WebBaseURL string
}

// SMTPConfig, geliştirmede Mailpit'e, gerekirse üretimde bir SMTP röle
// sunucusuna teslim için kullanılan ayarlar.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	// UseTLS, STARTTLS ile şifreli teslimi zorlar. Mailpit'te kapalıdır.
	UseTLS bool
}
