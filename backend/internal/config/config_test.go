package config

import (
	"os"
	"testing"
	"time"
)

func TestConfigLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected successful config load with defaults, got: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}

	if cfg.Pipeline.MaxConcurrentBatches != 10 {
		t.Errorf("expected default MaxConcurrentBatches 10, got %d", cfg.Pipeline.MaxConcurrentBatches)
	}

	if cfg.Pipeline.StageTimeout != 5*time.Minute {
		t.Errorf("expected default StageTimeout 5m, got %v", cfg.Pipeline.StageTimeout)
	}

	if cfg.Pipeline.RingBufferSizeKB != 2048 {
		t.Errorf("expected default RingBufferSizeKB 2048, got %d", cfg.Pipeline.RingBufferSizeKB)
	}

	if !cfg.Security.EnforceStrictSchema {
		t.Errorf("expected default EnforceStrictSchema true")
	}

	if cfg.AI.GeminiModel != "gemini-2.5-flash" {
		t.Errorf("expected default GeminiModel gemini-2.5-flash, got %s", cfg.AI.GeminiModel)
	}

	if cfg.Matching.ContractFeeRate != "0.0200" {
		t.Errorf("expected default ContractFeeRate 0.0200, got %s", cfg.Matching.ContractFeeRate)
	}

	if cfg.AutoSeedOnEmpty {
		t.Errorf("expected default AutoSeedOnEmpty false")
	}
}

func TestConfigValidationErrors(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 99999}, // invalid port
		Pipeline: PipelineConfig{
			MaxConcurrentBatches: 0, // invalid batches
			StageTimeout:         -1 * time.Second,
			RingBufferSizeKB:     32, // < 64
		},
		Auth:     AuthConfig{APIKey: ""},
		Database: DatabaseConfig{URL: ""},
		MockMode: false,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation error for invalid port, got nil")
	}

	cfg.Server.Port = 8080
	err = cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation error for zero batches, got nil")
	}

	cfg.Pipeline.MaxConcurrentBatches = 5
	err = cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation error for negative timeout, got nil")
	}

	cfg.Pipeline.StageTimeout = 1 * time.Minute
	err = cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation error for ring buffer < 64 KB, got nil")
	}

	cfg.Pipeline.RingBufferSizeKB = 1024
	err = cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation error for empty AUTH_API_KEY, got nil")
	}

	cfg.Auth.APIKey = "valid-key"
	err = cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation error for empty DATABASE_URL when MOCK_MODE=false, got nil")
	}

	// Should pass if MOCK_MODE=true with empty DATABASE_URL
	cfg.MockMode = true
	err = cfg.Validate()
	if err != nil {
		t.Fatalf("expected validation success for empty DATABASE_URL when MOCK_MODE=true, got: %v", err)
	}
}

func TestConfigEnvOverrides(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("PIPELINE_MAX_CONCURRENT_BATCHES", "20")
	os.Setenv("GEMINI_MODEL", "gemini-2.5-pro")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("PIPELINE_MAX_CONCURRENT_BATCHES")
		os.Unsetenv("GEMINI_MODEL")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090 from env, got %d", cfg.Server.Port)
	}

	if cfg.Pipeline.MaxConcurrentBatches != 20 {
		t.Errorf("expected 20 concurrent batches from env, got %d", cfg.Pipeline.MaxConcurrentBatches)
	}

	if cfg.AI.GeminiModel != "gemini-2.5-pro" {
		t.Errorf("expected model gemini-2.5-pro from env, got %s", cfg.AI.GeminiModel)
	}
}

