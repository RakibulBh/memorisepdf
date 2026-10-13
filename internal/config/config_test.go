package config

import (
	"net/url"
	"strings"
	"testing"
)

func TestDatabaseURLBuildsFromParts(t *testing.T) {
	t.Setenv("DB_HOST", "db")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_NAME", "unmemorize")
	t.Setenv("DB_USER", "unmemorize")
	t.Setenv("DB_PASSWORD", "plain-password")

	got := databaseURL()

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("the built string is not a valid url: %v", err)
	}
	if parsed.Scheme != "postgres" {
		t.Errorf("scheme = %q, want postgres", parsed.Scheme)
	}
	if parsed.Host != "db:5432" {
		t.Errorf("host = %q, want db:5432", parsed.Host)
	}
	if parsed.Path != "/unmemorize" {
		t.Errorf("path = %q, want /unmemorize", parsed.Path)
	}

	user := parsed.User.Username()
	password, _ := parsed.User.Password()
	if user != "unmemorize" {
		t.Errorf("user = %q, want unmemorize", user)
	}
	if password != "plain-password" {
		t.Errorf("password = %q, want it to survive the round trip", password)
	}
	if parsed.Query().Get("sslmode") != "disable" {
		t.Errorf("sslmode = %q, want disable", parsed.Query().Get("sslmode"))
	}
}

// The whole point of building the string here is that the password no longer
// has to be URL-safe. A password with the characters that would otherwise break
// a hand-written connection string has to survive.
func TestDatabaseURLEscapesAwkwardPasswords(t *testing.T) {
	for _, password := range []string{
		"p@ssw0rd",
		"has/slash",
		"has#hash",
		"has?question",
		"has:colon",
		"has%percent",
		"everything@/#?:%&=+ ",
	} {
		t.Setenv("DB_PASSWORD", password)
		t.Setenv("DB_HOST", "db")

		built := databaseURL()

		parsed, err := url.Parse(built)
		if err != nil {
			t.Errorf("password %q produced an unparseable url %q: %v", password, built, err)
			continue
		}

		got, set := parsed.User.Password()
		if !set {
			t.Errorf("password %q was lost entirely", password)
			continue
		}
		if got != password {
			t.Errorf("password %q came back as %q", password, got)
		}

		// The host must not be swallowed by an unescaped character in the
		// password, which is the failure a hand-written string would hit.
		if parsed.Host != "db:5432" {
			t.Errorf("password %q corrupted the host: got %q", password, parsed.Host)
		}
	}
}

// An explicit DATABASE_URL is how tests, CI and an external database point the
// app somewhere else, so it has to win.
func TestDatabaseURLPrefersAnExplicitValue(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://elsewhere/db?sslmode=require")
	t.Setenv("DB_PASSWORD", "ignored")

	if got := databaseURL(); got != "postgres://elsewhere/db?sslmode=require" {
		t.Errorf("databaseURL() = %q, want the explicit DATABASE_URL", got)
	}
}

// With nothing configured there is no connection string, which is what makes
// Load report a missing setting rather than dialling a default database.
func TestDatabaseURLIsEmptyWithoutAPassword(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_PASSWORD", "")

	if got := databaseURL(); got != "" {
		t.Errorf("databaseURL() = %q, want empty", got)
	}
}

func TestLoadReportsEverythingMissingAtOnce(t *testing.T) {
	for _, key := range []string{
		"DATABASE_URL", "DB_PASSWORD", "POSTGRES_PASSWORD",
		"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "GEMINI_API_KEY",
	} {
		t.Setenv(key, "")
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load succeeded with nothing configured, want an error")
	}

	// One message naming all of them beats being told about them one restart
	// at a time.
	for _, want := range []string{"DB_PASSWORD", "GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "GEMINI_API_KEY"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %s: %v", want, err)
		}
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DB_PASSWORD", "x")
	t.Setenv("GOOGLE_CLIENT_ID", "x")
	t.Setenv("GOOGLE_CLIENT_SECRET", "x")
	t.Setenv("GEMINI_API_KEY", "x")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Gemini.Model != "gemini-3.6-flash" {
		t.Errorf("model = %q, want gemini-3.6-flash", cfg.Gemini.Model)
	}
	if cfg.Limits.MaxFileSize != 100<<20 {
		t.Errorf("MaxFileSize = %d, want 100MB", cfg.Limits.MaxFileSize)
	}
	// The sign-in callback has to default to the address Google is told about.
	if cfg.Google.RedirectURL != cfg.AppURL+"/auth/google/callback" {
		t.Errorf("RedirectURL = %q, want it derived from AppURL", cfg.Google.RedirectURL)
	}
	// Cookies must only be insecure in development.
	if cfg.Session.Secure {
		t.Error("Session.Secure = true in development, want false")
	}

	t.Setenv("ENV", "production")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Session.Secure {
		t.Error("Session.Secure = false in production, want true")
	}
}

// A trailing slash on a configured URL must not survive into a built address.
func TestLoadTrimsTrailingSlashes(t *testing.T) {
	t.Setenv("DB_PASSWORD", "x")
	t.Setenv("GOOGLE_CLIENT_ID", "x")
	t.Setenv("GOOGLE_CLIENT_SECRET", "x")
	t.Setenv("GEMINI_API_KEY", "x")
	t.Setenv("APP_URL", "https://example.com/")
	t.Setenv("WEB_URL", "https://example.com/")
	t.Setenv("TIKA_URL", "http://tika:9998/")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.AppURL != "https://example.com" {
		t.Errorf("AppURL = %q, want no trailing slash", cfg.AppURL)
	}
	if cfg.Google.RedirectURL != "https://example.com/auth/google/callback" {
		t.Errorf("RedirectURL = %q, want a single slash before auth", cfg.Google.RedirectURL)
	}
	if cfg.Tika.URL != "http://tika:9998" {
		t.Errorf("Tika.URL = %q, want no trailing slash", cfg.Tika.URL)
	}
}

// LOG_FILE= is how the file sink is turned off, so an empty value must not fall
// back to the default name.
func TestEmptyLogFileDisablesTheFile(t *testing.T) {
	t.Setenv("DB_PASSWORD", "x")
	t.Setenv("GOOGLE_CLIENT_ID", "x")
	t.Setenv("GOOGLE_CLIENT_SECRET", "x")
	t.Setenv("GEMINI_API_KEY", "x")
	t.Setenv("LOG_FILE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Log.File != "" {
		t.Errorf("Log.File = %q, want empty", cfg.Log.File)
	}
}

// On the server only POSTGRES_PASSWORD is set, because that is the name the
// database container requires. The app has to accept it so the password is
// configured once rather than under two names.
func TestDatabaseURLAcceptsPostgresPassword(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("POSTGRES_PASSWORD", "from-compose")
	t.Setenv("DB_HOST", "db")

	parsed, err := url.Parse(databaseURL())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if password, _ := parsed.User.Password(); password != "from-compose" {
		t.Errorf("password = %q, want from-compose", password)
	}
}

// DB_PASSWORD is the app's own name for it, so it has to win when both are set.
func TestDBPasswordWinsOverPostgresPassword(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_PASSWORD", "preferred")
	t.Setenv("POSTGRES_PASSWORD", "fallback")

	parsed, err := url.Parse(databaseURL())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if password, _ := parsed.User.Password(); password != "preferred" {
		t.Errorf("password = %q, want preferred", password)
	}
}
