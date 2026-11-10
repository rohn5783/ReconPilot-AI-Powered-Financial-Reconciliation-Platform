package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the strictly typed, validated runtime configuration for ReconPilot.
type Config struct {
	Environment string         `mapstructure:"ENVIRONMENT"`
	Server      ServerConfig   `mapstructure:",squash"`
	Database    DatabaseConfig `mapstructure:",squash"`
	Redis       RedisConfig    `mapstructure:",squash"`
	Auth        AuthConfig     `mapstructure:",squash"`
	AI          AIConfig       `mapstructure:",squash"`
	Pipeline    PipelineConfig `mapstructure:",squash"`
	Logging     LoggingConfig  `mapstructure:",squash"`
}

type ServerConfig struct {
	Host               string        `mapstructure:"SERVER_HOST"`
	Port               int           `mapstructure:"SERVER_PORT"`
	ReadTimeout        time.Duration `mapstructure:"SERVER_READ_TIMEOUT"`
	WriteTimeout       time.Duration `mapstructure:"SERVER_WRITE_TIMEOUT"`
	IdleTimeout        time.Duration `mapstructure:"SERVER_IDLE_TIMEOUT"`
	CORSAllowedOrigins []string      `mapstructure:"CORS_ALLOWED_ORIGINS"`
}

type DatabaseConfig struct {
	URI      string        `mapstructure:"MONGODB_URI"`
	Database string        `mapstructure:"MONGODB_DATABASE"`
	Timeout  time.Duration `mapstructure:"MONGODB_TIMEOUT"`
	URL      string        `mapstructure:"DATABASE_URL"`
}

type RedisConfig struct {
	URL      string `mapstructure:"REDIS_URL"`
	Password string `mapstructure:"REDIS_PASSWORD"`
	DB       int    `mapstructure:"REDIS_DB"`
}

type AuthConfig struct {
	JWTSecret    string `mapstructure:"JWT_SECRET"`
	MasterAPIKey string `mapstructure:"MASTER_API_KEY"`
}

type AIConfig struct {
	GeminiAPIKey string  `mapstructure:"GEMINI_API_KEY"`
	GeminiModel  string  `mapstructure:"GEMINI_MODEL"`
	MaxTokens    int     `mapstructure:"GEMINI_MAX_TOKENS"`
	Temperature  float64 `mapstructure:"GEMINI_TEMPERATURE"`
}

type PipelineConfig struct {
	MaxConcurrentScans int           `mapstructure:"MAX_CONCURRENT_SCANS"`
	StageTimeout       time.Duration `mapstructure:"STAGE_TIMEOUT"`
	GlobalScanTimeout  time.Duration `mapstructure:"GLOBAL_SCAN_TIMEOUT"`
	DryRunDefault      bool          `mapstructure:"DRY_RUN_DEFAULT"`
	SubfinderBin       string        `mapstructure:"SUBFINDER_BIN"`
	HttpxBin           string        `mapstructure:"HTTPX_BIN"`
	NmapBin            string        `mapstructure:"NMAP_BIN"`
	NucleiBin          string        `mapstructure:"NUCLEI_BIN"`
	RingBufferSizeKB   int           `mapstructure:"RING_BUFFER_SIZE_KB"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"LOG_LEVEL"`
	Format string `mapstructure:"LOG_FORMAT"`
}

// Load loads configuration from environment variables, optional .env files, and sane defaults.
func Load() (*Config, error) {
	v := viper.New()

	v.SetDefault("ENVIRONMENT", "development")

	// Server defaults
	v.SetDefault("SERVER_HOST", "0.0.0.0")
	v.SetDefault("SERVER_PORT", 8080)
	v.SetDefault("SERVER_READ_TIMEOUT", 30*time.Second)
	v.SetDefault("SERVER_WRITE_TIMEOUT", 60*time.Second)
	v.SetDefault("SERVER_IDLE_TIMEOUT", 120*time.Second)
	v.SetDefault("CORS_ALLOWED_ORIGINS", []string{"http://localhost:5173", "http://localhost:3000", "http://127.0.0.1:5173"})

	// Database defaults (MongoDB)
	v.SetDefault("MONGODB_URI", "mongodb://localhost:27017")
	v.SetDefault("MONGODB_DATABASE", "reconpilot")
	v.SetDefault("MONGODB_TIMEOUT", 10*time.Second)
	v.SetDefault("DATABASE_URL", "mongodb://localhost:27017")

	// Redis defaults
	v.SetDefault("REDIS_URL", "redis://localhost:6379/0")
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("REDIS_DB", 0)

	// Auth defaults
	v.SetDefault("JWT_SECRET", "reconpilot-hardened-jwt-secret-replace-in-prod-32chars")
	v.SetDefault("MASTER_API_KEY", "reconpilot_adm_live_9f8d7c6b5a4e3d2c1")

	// AI defaults
	v.SetDefault("GEMINI_API_KEY", "")
	v.SetDefault("GEMINI_MODEL", "gemini-2.5-flash")
	v.SetDefault("GEMINI_MAX_TOKENS", 8192)
	v.SetDefault("GEMINI_TEMPERATURE", 0.2)

	// Pipeline defaults
	v.SetDefault("MAX_CONCURRENT_SCANS", 4)
	v.SetDefault("STAGE_TIMEOUT", 15*time.Minute)
	v.SetDefault("GLOBAL_SCAN_TIMEOUT", 45*time.Minute)
	v.SetDefault("DRY_RUN_DEFAULT", false)
	v.SetDefault("SUBFINDER_BIN", "subfinder")
	v.SetDefault("HTTPX_BIN", "httpx")
	v.SetDefault("NMAP_BIN", "nmap")
	v.SetDefault("NUCLEI_BIN", "nuclei")
	v.SetDefault("RING_BUFFER_SIZE_KB", 1024)

	// Logging defaults
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_FORMAT", "json")

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Optionally read local .env file if present
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) && !os.IsNotExist(err) {
			// Non-critical if file not found; critical if invalid format
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Override from direct os.Getenv for essential keys
	if envKey := os.Getenv("GEMINI_API_KEY"); envKey != "" {
		cfg.AI.GeminiAPIKey = envKey
	}
	if mongoURI := os.Getenv("MONGODB_URI"); mongoURI != "" {
		cfg.Database.URI = mongoURI
	}
	if mongoDB := os.Getenv("MONGODB_DATABASE"); mongoDB != "" {
		cfg.Database.Database = mongoDB
	}
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.Database.URL = dbURL
		if cfg.Database.URI == "" {
			cfg.Database.URI = dbURL
		}
	}
	if cfg.Database.URI == "" && cfg.Database.URL != "" {
		cfg.Database.URI = cfg.Database.URL
	}
	if cfg.Database.Database == "" {
		cfg.Database.Database = "reconpilot"
	}
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		cfg.Redis.URL = redisURL
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// Validate ensures all configuration invariants are satisfied.
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid SERVER_PORT: %d (must be 1-65535)", c.Server.Port)
	}

	if strings.TrimSpace(c.Database.URI) == "" && strings.TrimSpace(c.Database.URL) == "" {
		return errors.New("MONGODB_URI (or DATABASE_URL) must not be empty")
	}

	if strings.TrimSpace(c.Auth.JWTSecret) == "" {
		return errors.New("JWT_SECRET must not be empty")
	}

	if len(c.Auth.JWTSecret) < 16 && c.Environment == "production" {
		return errors.New("JWT_SECRET must be at least 16 characters in production")
	}

	if c.Pipeline.MaxConcurrentScans < 1 {
		return errors.New("MAX_CONCURRENT_SCANS must be at least 1")
	}

	if c.Pipeline.StageTimeout <= 0 {
		return errors.New("STAGE_TIMEOUT must be positive")
	}

	if c.Pipeline.RingBufferSizeKB < 64 {
		c.Pipeline.RingBufferSizeKB = 64
	}

	return nil
}

