package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env     string
	AppPort string
	// FrontendBaseURL is the frontend origin (e.g. https://www.finishlinebolivia.com)
	// used to build absolute links in emails (the logo). No default: it must be
	// supplied per environment via FRONTEND_BASE_URL.
	FrontendBaseURL string
	DB              DBConfig
	Auth            AuthConfig
	Sanity          SanityConfig
	Email           EmailConfig
	// ServiceSecret authenticates the frontend BFF (astro-finish-line-frontend)
	// when it calls internal-only endpoints on this API — currently just
	// POST /api/v1/registrations. It is a separate secret from
	// Sanity.WebhookSecret (a different caller, a different route) but is
	// checked with the same constant-time-compare pattern.
	ServiceSecret string
}

// SanityConfig holds the shared secret for the inbound Sanity webhook
// (`/webhooks/sanity`) that syncs the local race snapshot. Sanity replaced
// Strapi as the CMS of record; this config replaced the former
// StrapiConfig/STRAPI_WEBHOOK_SECRET 1:1.
type SanityConfig struct {
	WebhookSecret string
}

type EmailConfig struct {
	ResendAPIKey string
	From         string
}

type AuthConfig struct {
	JWTSecret  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type DBConfig struct {
	Host     string
	User     string
	Password string
	Name     string
	Port     string
	SSLMode  string
}

// DSN builds a URL connection string. Using net/url means a password with
// spaces or special characters is percent-encoded correctly, instead of
// breaking the space-delimited key=value format.
func (d DBConfig) DSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(d.User, d.Password),
		Host:     net.JoinHostPort(d.Host, d.Port),
		Path:     d.Name,
		RawQuery: url.Values{"sslmode": {d.SSLMode}}.Encode(),
	}
	return u.String()
}

func Load() (*Config, error) {
	if err := loadDotEnv(); err != nil {
		return nil, err
	}

	// The frontend origin differs per environment and is never guessed here:
	// a wrong value silently ships broken links in emails, so an unset
	// variable fails at startup instead.
	frontendBaseURL := strings.TrimRight(getEnv("FRONTEND_BASE_URL", ""), "/")
	if frontendBaseURL == "" {
		return nil, errors.New("FRONTEND_BASE_URL is required")
	}

	return &Config{
		Env:             getEnv("APP_ENV", "development"),
		AppPort:         getEnv("APP_PORT", "8080"),
		FrontendBaseURL: frontendBaseURL,
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "finishline"),
			Port:     getEnv("DB_PORT", "5432"),
			SSLMode:  getEnv("DB_SSLMODE", "require"),
		},
		Auth: AuthConfig{
			JWTSecret:  getEnv("JWT_SECRET", "dev-insecure-secret-change-me"),
			AccessTTL:  getEnvDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTTL: getEnvDuration("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		},
		Sanity: SanityConfig{
			WebhookSecret: getEnv("SANITY_WEBHOOK_SECRET", "dev-insecure-webhook-secret"),
		},
		Email: EmailConfig{
			ResendAPIKey: getEnv("RESEND_API_KEY", ""),
			From:         getEnv("EMAIL_FROM", "no-reply@finishline.dev"),
		},
		ServiceSecret: getEnv("SERVICE_SECRET", "dev-insecure-service-secret"),
	}, nil
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func loadDotEnv() error {
	if _, err := os.Stat(".env"); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err := godotenv.Load(); err != nil {
		return fmt.Errorf("loading .env: %w", err)
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// getEnvDuration parses a Go duration string (e.g. "15m", "168h"). An unset
// or unparseable value falls back to the default.
func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
