package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	Redis        RedisConfig
	JWT          JWTConfig
	Auth         AuthConfig
	Upload       UploadConfig
	RateLimit    RateLimitConfig
	Cache        CacheConfig
	Pagination   PaginationConfig
	CORS         CORSConfig
	Subscription SubscriptionConfig
	Payment      PaymentConfig
	KYC          KYCConfig
	FaceService  FaceServiceConfig
	Admin        AdminConfig
}

type ServerConfig struct {
	Port     string
	Env      string
	LogLevel string
}

type DatabaseConfig struct {
	URL          string
	MaxConns     int32
	MinConns     int32
	MaxConnLife  time.Duration
	MaxConnIdle  time.Duration
	HealthCheck  time.Duration
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret             string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

type AuthConfig struct {
	GoogleClientID string
	AppleTeamID    string
	AppleBundleID  string
	AppleKeyID     string
	AppleKeyPath   string
}

type UploadConfig struct {
	MaxFileSize    int64
	MaxFiles       int
	Dir            string
	AllowedMIME    []string
	KYCMaxFileSize int64
}

type RateLimitConfig struct {
	Window int
	Max    int
}

type CacheConfig struct {
	PostsTTL time.Duration
	GamesTTL time.Duration
	UserTTL  time.Duration
}

type PaginationConfig struct {
	DefaultLimit int
	MaxLimit     int
}

type CORSConfig struct {
	Origins []string
}

type SubscriptionConfig struct {
	PriceUZS    int64
	PriceUSD    int64
	DurationDays int
	GraceDays   int
}

type PaymentConfig struct {
	Payme PaymeConfig
	Click ClickConfig
	Xazna XaznaConfig
	Uzum  UzumConfig
}

type PaymeConfig struct {
	MerchantID  string
	SecretKey   string
	CallbackURL string
	WebhookURL  string
	TestMode    bool
}

type ClickConfig struct {
	MerchantID   string
	ServiceID    string
	SecretKey    string
	MerchantUser string
	WebhookURL   string
}

type XaznaConfig struct {
	BaseURL    string
	APIKey     string
	SecretKey  string
	WebhookURL string
}

type UzumConfig struct {
	BaseURL    string
	TerminalID string
	SecretKey  string
	WebhookURL string
}

type KYCConfig struct {
	FaceMatchThreshold float64
}

type FaceServiceConfig struct {
	Address string
	Timeout time.Duration
}

type AdminConfig struct {
	Email    string
	Password string
	Name     string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:     getEnv("PORT", "3000"),
			Env:      getEnv("NODE_ENV", "development"),
			LogLevel: getEnv("LOG_LEVEL", "debug"),
		},
		Database: DatabaseConfig{
			URL:         getEnvRequired("DATABASE_URL"),
			MaxConns:    int32(getEnvInt("DB_MAX_CONNS", 20)),
			MinConns:    int32(getEnvInt("DB_MIN_CONNS", 5)),
			MaxConnLife: time.Duration(getEnvInt("DB_MAX_CONN_LIFE_MINS", 60)) * time.Minute,
			MaxConnIdle: time.Duration(getEnvInt("DB_MAX_CONN_IDLE_MINS", 30)) * time.Minute,
			HealthCheck: time.Duration(getEnvInt("DB_HEALTH_CHECK_SECS", 60)) * time.Second,
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:             getEnvRequired("JWT_SECRET"),
			AccessTokenExpiry:  parseDuration(getEnv("JWT_ACCESS_EXPIRES", "1h")),
			RefreshTokenExpiry: parseDuration(getEnv("JWT_REFRESH_EXPIRES", "7d")),
		},
		Auth: AuthConfig{
			GoogleClientID: getEnv("GOOGLE_CLIENT_ID", ""),
			AppleTeamID:    getEnv("APPLE_TEAM_ID", ""),
			AppleBundleID:  getEnv("APPLE_BUNDLE_ID", ""),
			AppleKeyID:     getEnv("APPLE_KEY_ID", ""),
			AppleKeyPath:   getEnv("APPLE_KEY_PATH", ""),
		},
		Upload: UploadConfig{
			MaxFileSize:    int64(getEnvInt("MAX_FILE_SIZE", 5242880)),
			MaxFiles:       getEnvInt("MAX_FILES", 5),
			Dir:            getEnv("UPLOAD_DIR", "uploads"),
			AllowedMIME:    []string{"image/jpeg", "image/png", "image/webp", "image/heic", "image/heif"},
			KYCMaxFileSize: int64(getEnvInt("KYC_MAX_FILE_SIZE", 10485760)),
		},
		RateLimit: RateLimitConfig{
			Window: getEnvInt("RATE_LIMIT_WINDOW", 900000),
			Max:    getEnvInt("RATE_LIMIT_MAX", 100),
		},
		Cache: CacheConfig{
			PostsTTL: time.Duration(getEnvInt("CACHE_POSTS_TTL", 300)) * time.Second,
			GamesTTL: time.Duration(getEnvInt("CACHE_GAMES_TTL", 3600)) * time.Second,
			UserTTL:  time.Duration(getEnvInt("CACHE_USER_TTL", 600)) * time.Second,
		},
		Pagination: PaginationConfig{
			DefaultLimit: getEnvInt("DEFAULT_PAGE_LIMIT", 20),
			MaxLimit:     getEnvInt("MAX_PAGE_LIMIT", 100),
		},
		CORS: CORSConfig{
			Origins: strings.Split(getEnv("CORS_ORIGIN", "*"), ","),
		},
		Subscription: SubscriptionConfig{
			PriceUZS:     int64(getEnvInt("SUBSCRIPTION_PRICE_UZS", 10000)),
			PriceUSD:     int64(getEnvInt("SUBSCRIPTION_PRICE_USD", 1)),
			DurationDays: getEnvInt("SUBSCRIPTION_DURATION_DAYS", 30),
			GraceDays:    getEnvInt("GRACE_PERIOD_DAYS", 3),
		},
		Payment: PaymentConfig{
			Payme: PaymeConfig{
				MerchantID:  getEnv("PAYME_MERCHANT_ID", ""),
				SecretKey:   getEnv("PAYME_SECRET_KEY", ""),
				CallbackURL: getEnv("PAYME_CALLBACK_URL", ""),
				WebhookURL:  getEnv("PAYME_WEBHOOK_URL", ""),
				TestMode:    getEnvBool("PAYME_TEST_MODE", true),
			},
			Click: ClickConfig{
				MerchantID:   getEnv("CLICK_MERCHANT_ID", ""),
				ServiceID:    getEnv("CLICK_SERVICE_ID", ""),
				SecretKey:    getEnv("CLICK_SECRET_KEY", ""),
				MerchantUser: getEnv("CLICK_MERCHANT_USER", ""),
				WebhookURL:   getEnv("CLICK_WEBHOOK_URL", ""),
			},
			Xazna: XaznaConfig{
				BaseURL:    getEnv("XAZNA_BASE_URL", ""),
				APIKey:     getEnv("XAZNA_API_KEY", ""),
				SecretKey:  getEnv("XAZNA_SECRET_KEY", ""),
				WebhookURL: getEnv("XAZNA_WEBHOOK_URL", ""),
			},
			Uzum: UzumConfig{
				BaseURL:    getEnv("UZUM_BASE_URL", ""),
				TerminalID: getEnv("UZUM_TERMINAL_ID", ""),
				SecretKey:  getEnv("UZUM_SECRET_KEY", ""),
				WebhookURL: getEnv("UZUM_WEBHOOK_URL", ""),
			},
		},
		KYC: KYCConfig{
			FaceMatchThreshold: getEnvFloat("KYC_FACE_MATCH_THRESHOLD", 0.6),
		},
		FaceService: FaceServiceConfig{
			Address: getEnv("FACE_SERVICE_ADDRESS", "localhost:50051"),
			Timeout: time.Duration(getEnvInt("FACE_SERVICE_TIMEOUT_SECS", 30)) * time.Second,
		},
		Admin: AdminConfig{
			Email:    getEnv("ADMIN_EMAIL", ""),
			Password: getEnv("ADMIN_PASSWORD", ""),
			Name:     getEnv("ADMIN_NAME", "Super Admin"),
		},
	}

	if cfg.Server.Env == "production" {
		if cfg.JWT.Secret == "" {
			return nil, fmt.Errorf("JWT_SECRET is required in production")
		}
		if cfg.Auth.GoogleClientID == "" {
			return nil, fmt.Errorf("GOOGLE_CLIENT_ID is required in production")
		}
	}

	return cfg, nil
}

func (r RedisConfig) Addr() string {
	return r.Host + ":" + r.Port
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvRequired(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func getEnvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func parseDuration(s string) time.Duration {
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err == nil {
			return time.Duration(days) * 24 * time.Hour
		}
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Hour
	}
	return d
}
