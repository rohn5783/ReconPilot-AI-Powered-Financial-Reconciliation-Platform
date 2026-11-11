package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config encapsulates strictly typed, validated environment configuration for ReconPilot.
type Config struct {
	Pipeline        PipelineConfig `mapstructure:",squash"`
	Security        SecurityConfig `mapstructure:",squash"`
	Matching        MatchingConfig `mapstructure:",squash"`
	AI              AIConfig       `mapstructure:",squash"`
	Auth            AuthConfig     `mapstructure:",squash"`
	MockMode        bool           `mapstructure:"MOCK_MODE"`
	AutoSeedOnEmpty bool           `mapstructure:"AUTO_SEED_ON_EMPTY"`
	Database        DatabaseConfig `mapstructure:",squash"`
	Server          ServerConfig   `mapstructure:",squash"`
}

// MatchingConfig configures dynamic contract rates and matching thresholds.
type MatchingConfig struct {
	ContractFeeRate string `mapstructure:"CONTRACT_FEE_RATE"`
	ContractTaxRate string `mapstructure:"CONTRACT_TAX_RATE"`
	FeeTolerance    string `mapstructure:"FEE_TOLERANCE"`
}

// PipelineConfig controls concurrency, timeouts, and bounded streaming buffers.
type PipelineConfig struct {
	MaxConcurrentBatches int           `mapstructure:"PIPELINE_MAX_CONCURRENT_BATCHES"`
	StageTimeout         time.Duration `mapstructure:"PIPELINE_STAGE_TIMEOUT"`
	RingBufferSizeKB     int           `mapstructure:"PIPELINE_RING_BUFFER_SIZE_KB"`
}

// SecurityConfig controls ingestion validation rigor.
type SecurityConfig struct {
	EnforceStrictSchema bool `mapstructure:"SECURITY_ENFORCE_STRICT_SCHEMA"`
}

// AIConfig configures Google Gemini 2.5 Flash for discrepancy diagnosis.
type AIConfig struct {
	GeminiAPIKey string `mapstructure:"GEMINI_API_KEY"`
	GeminiModel  string `mapstructure:"GEMINI_MODEL"`
}

// AuthConfig defines API authentication keys.
type AuthConfig struct {
	APIKey string `mapstructure:"AUTH_API_KEY"`
}

// DatabaseConfig holds PostgreSQL relational connection settings.
type DatabaseConfig struct {
	URL string `mapstructure:"DATABASE_URL"`
}

// ServerConfig holds HTTP server listening settings.
type ServerConfig struct {
	Port int `mapstructure:"PORT"`
}

// Load loads configuration from environment variables, optional .env files, and defaults.
func Load() (*Config, error) {
	v := viper.New()

	// 1. Set explicit defaults conforming strictly to Part 1.1
	v.SetDefault("PIPELINE_MAX_CONCURRENT_BATCHES", 10)
	v.SetDefault("PIPELINE_STAGE_TIMEOUT", 5*time.Minute)
	v.SetDefault("PIPELINE_RING_BUFFER_SIZE_KB", 2048)

	v.SetDefault("SECURITY_ENFORCE_STRICT_SCHEMA", true)

	v.SetDefault("GEMINI_API_KEY", "")
	v.SetDefault("GEMINI_MODEL", "gemini-2.5-flash")

	v.SetDefault("AUTH_API_KEY", "")
	v.SetDefault("MOCK_MODE", false)
	v.SetDefault("AUTO_SEED_ON_EMPTY", false)
	v.SetDefault("CONTRACT_FEE_RATE", "0.0200")
	v.SetDefault("CONTRACT_TAX_RATE", "0.1800")
	v.SetDefault("FEE_TOLERANCE", "0.0100")

	v.SetDefault("DATABASE_URL", "postgres://reconpilot:reconpilot@localhost:5432/reconpilot?sslmode=disable")
	v.SetDefault("PORT", 8080)

	// 2. Read environment variables automatically
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 3. Optionally read local .env file
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) && !os.IsNotExist(err) {
			// Non-critical if file is absent, error if syntax is corrupted
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Direct OS env fallbacks for critical cloud injection
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		cfg.AI.GeminiAPIKey = apiKey
	}
	if geminiModel := os.Getenv("GEMINI_MODEL"); geminiModel != "" {
		cfg.AI.GeminiModel = geminiModel
	}
	if authKey := os.Getenv("AUTH_API_KEY"); authKey != "" {
		cfg.Auth.APIKey = authKey
	}
	if autoSeed := os.Getenv("AUTO_SEED_ON_EMPTY"); autoSeed != "" {
		cfg.AutoSeedOnEmpty = (autoSeed == "true" || autoSeed == "1")
	}
	if feeRate := os.Getenv("CONTRACT_FEE_RATE"); feeRate != "" {
		cfg.Matching.ContractFeeRate = feeRate
	}
	if taxRate := os.Getenv("CONTRACT_TAX_RATE"); taxRate != "" {
		cfg.Matching.ContractTaxRate = taxRate
	}
	if tol := os.Getenv("FEE_TOLERANCE"); tol != "" {
		cfg.Matching.FeeTolerance = tol
	}
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.Database.URL = dbURL
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// Validate verifies configuration bounds and invariants.
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid PORT: %d (must be between 1 and 65535)", c.Server.Port)
	}

	if c.Pipeline.MaxConcurrentBatches < 1 {
		return errors.New("PIPELINE_MAX_CONCURRENT_BATCHES must be at least 1")
	}

	if c.Pipeline.StageTimeout <= 0 {
		return errors.New("PIPELINE_STAGE_TIMEOUT must be positive")
	}

	if c.Pipeline.RingBufferSizeKB < 64 {
		return errors.New("PIPELINE_RING_BUFFER_SIZE_KB must be at least 64 KB")
	}

	if strings.TrimSpace(c.Database.URL) == "" && !c.MockMode {
		return errors.New("DATABASE_URL must not be empty when MOCK_MODE is false")
	}

	return nil
}

