// Package config reads every setting the app needs from the environment once at boot.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr    string
	Env     string
	AppURL  string
	WebURL  string
	DataDir string

	DatabaseURL string

	Google  Google
	Session Session
	Gemini  Gemini
	Tika    Tika
	Limits  Limits
	Log     Log
}

type Google struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type Session struct {
	CookieName string
	TTL        time.Duration
	Secure     bool
}

type Gemini struct {
	APIKey          string
	Model           string
	MaxOutputTokens int
	Timeout         time.Duration
}

type Tika struct {
	URL     string
	Timeout time.Duration
}

type Limits struct {
	MaxFileSize       int64
	UserStorageQuota  int64
	MaxFilesPerModule int
	GeneratePerHour   int
	MaxExtractedChars int
}

type Log struct {
	Level string
	File  string
}

// Load gathers all the settings from the environment and complains if a required one is missing.
func Load() (Config, error) {
	env := str("ENV", "development")
	appURL := strings.TrimSuffix(str("APP_URL", "http://localhost:8080"), "/")

	cfg := Config{
		Addr:        ":" + str("PORT", "8080"),
		Env:         env,
		AppURL:      appURL,
		WebURL:      strings.TrimSuffix(str("WEB_URL", "http://localhost:3000"), "/"),
		DataDir:     str("DATA_DIR", "./data"),
		DatabaseURL: str("DATABASE_URL", ""),
		Google: Google{
			ClientID:     str("GOOGLE_CLIENT_ID", ""),
			ClientSecret: str("GOOGLE_CLIENT_SECRET", ""),
			RedirectURL:  str("GOOGLE_REDIRECT_URL", appURL+"/auth/google/callback"),
		},
		Session: Session{
			CookieName: str("SESSION_COOKIE", "unmemorize_session"),
			TTL:        dur("SESSION_TTL", 30*24*time.Hour),
			Secure:     boolean("SESSION_SECURE", env != "development"),
		},
		Gemini: Gemini{
			APIKey:          str("GEMINI_API_KEY", ""),
			Model:           str("GEMINI_MODEL", "gemini-3.6-flash"),
			MaxOutputTokens: integer("GEMINI_MAX_OUTPUT_TOKENS", 32768),
			Timeout:         dur("GEMINI_TIMEOUT", 3*time.Minute),
		},
		Tika: Tika{
			URL:     strings.TrimSuffix(str("TIKA_URL", "http://tika:9998"), "/"),
			Timeout: dur("TIKA_TIMEOUT", 2*time.Minute),
		},
		Limits: Limits{
			MaxFileSize:       bytes("MAX_FILE_SIZE", 100<<20),
			UserStorageQuota:  bytes("USER_STORAGE_QUOTA", 500<<20),
			MaxFilesPerModule: integer("MAX_FILES_PER_MODULE", 50),
			GeneratePerHour:   integer("GENERATE_RATE_LIMIT", 20),
			MaxExtractedChars: integer("MAX_EXTRACTED_CHARS", 400_000),
		},
		Log: Log{
			Level: str("LOG_LEVEL", "info"),
			File:  str("LOG_FILE", "app.log"),
		},
	}

	var missing []string
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if cfg.Google.ClientID == "" {
		missing = append(missing, "GOOGLE_CLIENT_ID")
	}
	if cfg.Google.ClientSecret == "" {
		missing = append(missing, "GOOGLE_CLIENT_SECRET")
	}
	if cfg.Gemini.APIKey == "" {
		missing = append(missing, "GEMINI_API_KEY")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

// str reads a text setting, falling back to a default when it is not set.
func str(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// integer reads a whole-number setting, falling back when it is missing or unreadable.
func integer(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// bytes reads a size setting in bytes, falling back when it is missing or unreadable.
func bytes(key string, fallback int64) int64 {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

// boolean reads a true/false setting, falling back when it is missing or unreadable.
func boolean(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

// dur reads a time setting like "30m" or "720h", falling back when it is missing or unreadable.
func dur(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
