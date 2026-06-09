package config

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type CORSConfig struct {
	AllowedOrigins []string
}

type RateLimitConfig struct {
	GlobalRPS float64
	LoginRPS  float64
}

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

type AuthConfig struct {
	JWTSecret       []byte
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type Config struct {
	AppPort          string
	DatabaseURL      string
	ImageFilterURL   string
	TextFilterURL    string
	UploadsDir       string
	RegionsGeoJSONPath string
	EmailEnabled     bool
	SMTP             SMTPConfig
	Auth             AuthConfig
	CORS             CORSConfig
	RateLimit        RateLimitConfig
	MaxBodyBytes     string
	EmailWorkerCount int
}

func Load() (Config, error) {
	_ = godotenv.Load()

	appPort := strings.TrimSpace(os.Getenv("APP_PORT"))
	if appPort == "" {
		appPort = "8080"
	}

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}

	emailEnabled := parseBool(os.Getenv("EMAIL_ENABLED"))
	smtpPort, err := parseIntRequired("SMTP_PORT", os.Getenv("SMTP_PORT"))
	if err != nil && emailEnabled {
		return Config{}, err
	}

	workerCount := 2
	if v := strings.TrimSpace(os.Getenv("EMAIL_WORKERS")); v != "" {
		if n, convErr := strconv.Atoi(v); convErr == nil && n > 0 && n <= 32 {
			workerCount = n
		}
	}

	cfg := Config{
		AppPort:        appPort,
		DatabaseURL:    dbURL,
		ImageFilterURL: strings.TrimSpace(os.Getenv("IMAGE_FILTER_URL")),
		TextFilterURL:  strings.TrimSpace(os.Getenv("TEXT_FILTER_URL")),
		UploadsDir:     strings.TrimSpace(os.Getenv("UPLOADS_DIR")),
		RegionsGeoJSONPath: strings.TrimSpace(os.Getenv("REGIONS_GEOJSON_PATH")),
		EmailEnabled:   emailEnabled,
		SMTP: SMTPConfig{
			Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Port:     smtpPort,
			User:     strings.TrimSpace(os.Getenv("SMTP_USER")),
			Password: strings.TrimSpace(os.Getenv("SMTP_PASSWORD")),
			From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
		},
		Auth: AuthConfig{
			JWTSecret:       []byte(jwtSecret),
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 7 * 24 * time.Hour,
		},
		CORS: CORSConfig{
			AllowedOrigins: []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		},
		RateLimit: RateLimitConfig{
			GlobalRPS: 10,
			LoginRPS:  1,
		},
		MaxBodyBytes:     "10M",
		EmailWorkerCount: workerCount,
	}

	if cfg.UploadsDir == "" {
		cfg.UploadsDir = "uploads"
	}
	if cfg.RegionsGeoJSONPath == "" {
		cfg.RegionsGeoJSONPath = "AL2-AL8/russia_al2al8.geojson"
	}

	if emailEnabled {
		if cfg.SMTP.Host == "" || cfg.SMTP.User == "" || cfg.SMTP.Password == "" || cfg.SMTP.From == "" || cfg.SMTP.Port == 0 {
			return Config{}, errors.New("SMTP_* vars are required when EMAIL_ENABLED=true")
		}
	}

	return cfg, nil
}

func parseBool(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	return v == "1" || v == "true" || v == "yes" || v == "y" || v == "on"
}

func parseIntRequired(name, v string) (int, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, fmt.Errorf("%s is required", name)
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 || n > 65535 {
		return 0, fmt.Errorf("%s is invalid", name)
	}
	return n, nil
}

func (c CORSConfig) AddHeaders(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	allowed := false
	for _, o := range c.AllowedOrigins {
		if o == origin {
			allowed = true
			break
		}
	}
	if !allowed {
		return false
	}

	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	return true
}
