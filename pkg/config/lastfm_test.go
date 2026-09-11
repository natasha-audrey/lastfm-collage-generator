package config

import (
	"os"
	"testing"
)

func TestGetEnv_ReturnsValueWhenSet(t *testing.T) {
	t.Setenv("TEST_GET_ENV_KEY", "from-env")

	got := getEnv("TEST_GET_ENV_KEY", "fallback")
	if got != "from-env" {
		t.Fatalf("getEnv() = %q, want %q", got, "from-env")
	}
}

func TestGetEnv_ReturnsEmptyStringWhenSetToEmpty(t *testing.T) {
	t.Setenv("TEST_GET_ENV_EMPTY", "")

	got := getEnv("TEST_GET_ENV_EMPTY", "fallback")
	if got != "" {
		t.Fatalf("getEnv() = %q, want empty string", got)
	}
}

func TestGetEnv_ReturnsFallbackWhenUnset(t *testing.T) {
	const key = "TEST_GET_ENV_UNSET"
	t.Setenv(key, "temporary")
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Unsetenv(%q): %v", key, err)
	}

	got := getEnv(key, "fallback")
	if got != "fallback" {
		t.Fatalf("getEnv() = %q, want %q", got, "fallback")
	}
}

func TestLastFmConfigInit_ReadsEnvironmentVariables(t *testing.T) {
	t.Setenv("APPLICATION_NAME", "collage-gen")
	t.Setenv("API_KEY", "api-key")
	t.Setenv("SHARED_SECRET", "shared-secret")
	t.Setenv("REGISTERED_TO", "natasha")
	t.Setenv("BASE_URL", "https://example.test/2.0")

	var cfg LastFmConfig
	cfg.Init()

	if cfg.ApplicationName != "collage-gen" {
		t.Errorf("ApplicationName = %q, want %q", cfg.ApplicationName, "collage-gen")
	}
	if cfg.APIKey != "api-key" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "api-key")
	}
	if cfg.SharedSecret != "shared-secret" {
		t.Errorf("SharedSecret = %q, want %q", cfg.SharedSecret, "shared-secret")
	}
	if cfg.RegisteredTo != "natasha" {
		t.Errorf("RegisteredTo = %q, want %q", cfg.RegisteredTo, "natasha")
	}
	if cfg.BaseURL != "https://example.test/2.0" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://example.test/2.0")
	}
}

func TestLastFmConfigInit_UsesDefaultBaseURLWhenUnset(t *testing.T) {
	t.Setenv("APPLICATION_NAME", "")
	t.Setenv("API_KEY", "")
	t.Setenv("SHARED_SECRET", "")
	t.Setenv("REGISTERED_TO", "")
	t.Setenv("BASE_URL", "temporary")
	if err := os.Unsetenv("BASE_URL"); err != nil {
		t.Fatalf("Unsetenv(BASE_URL): %v", err)
	}

	var cfg LastFmConfig
	cfg.Init()

	const wantBaseURL = "http://ws.audioscrobbler.com/2.0"
	if cfg.BaseURL != wantBaseURL {
		t.Fatalf("BaseURL = %q, want default %q", cfg.BaseURL, wantBaseURL)
	}
	if cfg.ApplicationName != "" {
		t.Errorf("ApplicationName = %q, want empty string", cfg.ApplicationName)
	}
	if cfg.APIKey != "" {
		t.Errorf("APIKey = %q, want empty string", cfg.APIKey)
	}
	if cfg.SharedSecret != "" {
		t.Errorf("SharedSecret = %q, want empty string", cfg.SharedSecret)
	}
	if cfg.RegisteredTo != "" {
		t.Errorf("RegisteredTo = %q, want empty string", cfg.RegisteredTo)
	}
}
