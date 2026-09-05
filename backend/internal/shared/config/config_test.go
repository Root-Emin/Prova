package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Defaults(t *testing.T) {
	cfg := Load()

	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "masterfabric", cfg.Database.User)
	assert.Equal(t, "localhost", cfg.Redis.Host)
	assert.Equal(t, 6379, cfg.Redis.Port)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
}

func TestLoad_EnvironmentOverrides(t *testing.T) {
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("DB_HOST", "db.example.com")
	defer os.Unsetenv("SERVER_PORT")
	defer os.Unsetenv("DB_HOST")

	cfg := Load()
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "db.example.com", cfg.Database.Host)
}

func TestLoad_ResendVerificationSettings(t *testing.T) {
	t.Setenv("RESEND_API_KEY", "test-key")
	t.Setenv("RESEND_FROM_EMAIL", "Prova <onboarding@resend.dev>")
	t.Setenv("EMAIL_VERIFICATION_OTP_PEPPER", "verification-pepper")

	cfg := Load()

	assert.Equal(t, "test-key", cfg.Email.Resend.APIKey)
	assert.Equal(t, "Prova <onboarding@resend.dev>", cfg.Email.FromAddress)
	assert.Equal(t, "verification-pepper", cfg.EmailVerification.Pepper)
	assert.Equal(t, "verification-pepper", cfg.Auth.CodePepper, "legacy login config keeps the compatibility fallback")
	assert.Equal(t, 5*time.Minute, cfg.EmailVerification.OTPTTL)
	assert.Equal(t, 5, cfg.EmailVerification.MaxAttempts)
	assert.Equal(t, time.Minute, cfg.EmailVerification.ResendCooldown)
}

func TestLoad_DBPoolInt32Bounds(t *testing.T) {
	os.Setenv("DB_MAX_CONNS", "50")
	os.Setenv("DB_MIN_CONNS", "2147483648")
	defer os.Unsetenv("DB_MAX_CONNS")
	defer os.Unsetenv("DB_MIN_CONNS")

	cfg := Load()
	assert.Equal(t, int32(50), cfg.Database.MaxConns)
	assert.Equal(t, int32(5), cfg.Database.MinConns)
}

func TestDatabaseConfig_DSN(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "pass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}
	expected := "postgres://user:pass@localhost:5432/testdb?sslmode=disable"
	assert.Equal(t, expected, cfg.DSN())
}

func TestDatabaseConfig_DSN_EscapesSpecialCharacters(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "user@domain",
		Password: "p@ss:w?rd#",
		DBName:   "testdb",
		SSLMode:  "require",
	}
	dsn := cfg.DSN()
	assert.Contains(t, dsn, "postgres://")
	assert.Contains(t, dsn, "sslmode=require")
	assert.NotContains(t, dsn, "p@ss:w?rd#")
}

func TestRedisConfig_Addr(t *testing.T) {
	cfg := RedisConfig{Host: "redis.local", Port: 6380}
	assert.Equal(t, "redis.local:6380", cfg.Addr())
}
