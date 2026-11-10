package config

import (
	"os"
	"testing"
)

func TestConfigLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected successful config load with defaults, got: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}

	if cfg.Pipeline.MaxConcurrentScans != 4 {
		t.Errorf("expected default MaxConcurrentScans 4, got %d", cfg.Pipeline.MaxConcurrentScans)
	}

	if cfg.AI.GeminiModel != "gemini-2.5-flash" {
		t.Errorf("expected default model gemini-2.5-flash, got %s", cfg.AI.GeminiModel)
	}
}

func TestConfigValidation(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: 70000, // Invalid port > 65535
		},
		Database: DatabaseConfig{
			URL: "postgres://localhost/test",
		},
		Auth: AuthConfig{
			JWTSecret: "test-secret-12345",
		},
		Pipeline: PipelineConfig{
			MaxConcurrentScans: 2,
			StageTimeout:       10,
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Errorf("expected validation failure for port 70000, got nil")
	}

	cfg.Server.Port = 8080
	cfg.Database.URL = ""
	cfg.Database.URI = ""
	if err := cfg.Validate(); err == nil {
		t.Errorf("expected validation failure for empty database URL, got nil")
	}
}

func TestConfigEnvOverride(t *testing.T) {
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("GEMINI_MODEL", "gemini-1.5-pro")
	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("GEMINI_MODEL")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090 from env, got %d", cfg.Server.Port)
	}

	if cfg.AI.GeminiModel != "gemini-1.5-pro" {
		t.Errorf("expected model gemini-1.5-pro from env, got %s", cfg.AI.GeminiModel)
	}
}

