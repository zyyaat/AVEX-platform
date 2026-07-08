// Package config provides typed configuration loading from environment variables.
//
// The Config struct holds all settings for the entire platform. Fields are
// grouped by subsystem. Only the fields needed for the current phase are
// actively consumed; the rest are loaded and available for future modules.
//
// Design decisions:
//   - No external config library (viper, envconfig, etc.) — stdlib only.
//   - Required vars are validated at Load() time; missing required vars
//     cause Load() to return an error.
//   - All durations use Go duration strings (e.g. "30m", "24h", "500ms").
//   - Slices use comma-separated values in env vars.
package config

import (
        "fmt"
        "os"
        "strconv"
        "strings"
        "time"
)

// Environment represents the deployment environment.
type Environment string

const (
        EnvDevelopment Environment = "development"
        EnvStaging     Environment = "staging"
        EnvProduction  Environment = "production"
)

// Config holds all platform configuration.
type Config struct {
        App      AppConfig
        Database DatabaseConfig
        Redis    RedisConfig
        JWT      JWTConfig
        Bcrypt   BcryptConfig
        OTEL     OTELConfig
        CORS     CORSConfig
        Secrets  SecretsConfig
        Outbox   OutboxConfig
}

// AppConfig holds application-level settings.
type AppConfig struct {
        Name       string
        Env        Environment
        Port       string
        InstanceID string
        LogLevel   string // debug | info | warn | error
        LogFormat  string // json | text
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
        URL             string
        MaxConns        int32
        MinConns        int32
        MaxConnLifetime time.Duration
        MaxConnIdleTime time.Duration
}

// RedisConfig holds Redis connection settings.
// Used by the event bus, cache, and presence (not in Phase 1 but defined here).
type RedisConfig struct {
        URL      string
        Password string
        PoolSize int
}

// JWTConfig holds JWT signing settings.
type JWTConfig struct {
        Secret     string
        Issuer     string
        AccessTTL  time.Duration
        RefreshTTL time.Duration
}

// BcryptConfig holds password hashing settings.
type BcryptConfig struct {
        Cost int
}

// OTELConfig holds OpenTelemetry settings.
// Not consumed in Phase 1 but defined for forward compatibility.
type OTELConfig struct {
        Exporter     string // stdout | otlp
        OTLPEndpoint string
        ServiceName  string
        SamplerRatio float64
}

// CORSConfig holds CORS settings.
type CORSConfig struct {
        AllowedOrigins   []string
        AllowedMethods   []string
        AllowedHeaders   []string
        AllowCredentials bool
}

// SecretsConfig holds the secret provider settings.
// Not consumed in Phase 1 (env provider is used by default).
type SecretsConfig struct {
        Provider   string // env | file | vault
        FilePath   string
        VaultAddr  string
        VaultToken string
}

// OutboxConfig holds the outbox publisher worker settings.
// Not consumed in Phase 1 but defined for forward compatibility.
type OutboxConfig struct {
        PollInterval   time.Duration
        BatchSize      int
        MaxRetries     int
        RetryBaseDelay time.Duration
}

// Load reads configuration from environment variables and validates required fields.
// Returns an error if any required variable is missing or invalid.
func Load() (*Config, error) {
        cfg := &Config{
                App: AppConfig{
                        Name:       getEnv("APP_NAME", "avex-backend"),
                        Env:        Environment(getEnv("APP_ENV", "development")),
                        Port:       getEnv("APP_PORT", "8080"),
                        InstanceID: getEnv("APP_INSTANCE_ID", "node-1"),
                        LogLevel:   getEnv("APP_LOG_LEVEL", "info"),
                        LogFormat:  getEnv("APP_LOG_FORMAT", "json"),
                },
                Database: DatabaseConfig{
                        URL:             os.Getenv("DATABASE_URL"),
                        MaxConns:        int32(getEnvInt("DATABASE_MAX_CONNS", 25)),
                        MinConns:        int32(getEnvInt("DATABASE_MIN_CONNS", 5)),
                        MaxConnLifetime: getEnvDuration("DATABASE_MAX_CONN_LIFETIME", 30*time.Minute),
                        MaxConnIdleTime: getEnvDuration("DATABASE_MAX_CONN_IDLE_TIME", 5*time.Minute),
                },
                Redis: RedisConfig{
                        URL:      getEnv("REDIS_URL", "redis://localhost:6379/0"),
                        Password: os.Getenv("REDIS_PASSWORD"),
                        PoolSize: getEnvInt("REDIS_POOL_SIZE", 20),
                },
                JWT: JWTConfig{
                        Secret:     os.Getenv("JWT_SECRET"),
                        Issuer:     getEnv("JWT_ISSUER", "avex"),
                        AccessTTL:  getEnvDuration("JWT_ACCESS_TTL", 24*time.Hour),
                        RefreshTTL: getEnvDuration("JWT_REFRESH_TTL", 720*time.Hour),
                },
                Bcrypt: BcryptConfig{
                        Cost: getEnvInt("BCRYPT_COST", 12),
                },
                OTEL: OTELConfig{
                        Exporter:     getEnv("OTEL_EXPORTER", "stdout"),
                        OTLPEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318"),
                        ServiceName:  getEnv("OTEL_SERVICE_NAME", "avex-backend"),
                        SamplerRatio: getEnvFloat("OTEL_SAMPLER_RATIO", 1.0),
                },
                CORS: CORSConfig{
                        AllowedOrigins:   getEnvSlice("CORS_ALLOWED_ORIGINS", []string{"*"}),
                        AllowedMethods:   getEnvSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
                        AllowedHeaders:   getEnvSlice("CORS_ALLOWED_HEADERS", []string{"Authorization", "Content-Type", "Accept", "X-Request-Id"}),
                        AllowCredentials: getEnvBool("CORS_ALLOW_CREDENTIALS", true),
                },
                Secrets: SecretsConfig{
                        Provider:   getEnv("SECRETS_PROVIDER", "env"),
                        FilePath:   os.Getenv("SECRETS_FILE_PATH"),
                        VaultAddr:  os.Getenv("VAULT_ADDR"),
                        VaultToken: os.Getenv("VAULT_TOKEN"),
                },
                Outbox: OutboxConfig{
                        PollInterval:   getEnvDuration("OUTBOX_POLL_INTERVAL", 500*time.Millisecond),
                        BatchSize:      getEnvInt("OUTBOX_BATCH_SIZE", 100),
                        MaxRetries:     getEnvInt("OUTBOX_MAX_RETRIES", 10),
                        RetryBaseDelay: getEnvDuration("OUTBOX_RETRY_BASE_DELAY", 1*time.Second),
                },
        }

        if err := cfg.validate(); err != nil {
                return nil, fmt.Errorf("config validation: %w", err)
        }

        return cfg, nil
}

// validate checks that required fields are present and valid.
func (c *Config) validate() error {
        var errs []error

        // Database URL is always required.
        if c.Database.URL == "" {
                errs = append(errs, fmt.Errorf("DATABASE_URL is required"))
        }

        // JWT secret is required and must be at least 32 chars in production.
        if c.JWT.Secret == "" {
                errs = append(errs, fmt.Errorf("JWT_SECRET is required"))
        } else if len(c.JWT.Secret) < 32 && c.App.Env == EnvProduction {
                errs = append(errs, fmt.Errorf("JWT_SECRET must be at least 32 characters in production"))
        }

        // Bcrypt cost must be in valid range (4-31).
        if c.Bcrypt.Cost < 4 || c.Bcrypt.Cost > 31 {
                errs = append(errs, fmt.Errorf("BCRYPT_COST must be between 4 and 31, got %d", c.Bcrypt.Cost))
        }

        // Environment must be a known value.
        switch c.App.Env {
        case EnvDevelopment, EnvStaging, EnvProduction:
                // ok
        default:
                errs = append(errs, fmt.Errorf("APP_ENV must be development|staging|production, got %q", c.App.Env))
        }

        // Log level must be a known value.
        switch c.App.LogLevel {
        case "debug", "info", "warn", "error":
                // ok
        default:
                errs = append(errs, fmt.Errorf("APP_LOG_LEVEL must be debug|info|warn|error, got %q", c.App.LogLevel))
        }

        // Log format must be a known value.
        switch c.App.LogFormat {
        case "json", "text":
                // ok
        default:
                errs = append(errs, fmt.Errorf("APP_LOG_FORMAT must be json|text, got %q", c.App.LogFormat))
        }

        if len(errs) > 0 {
                return errs[0]
        }
        return nil
}

// IsProduction returns true if the environment is production.
func (c *Config) IsProduction() bool {
        return c.App.Env == EnvProduction
}

// IsDevelopment returns true if the environment is development.
func (c *Config) IsDevelopment() bool {
        return c.App.Env == EnvDevelopment
}

// ----- env helpers -----

func getEnv(key, fallback string) string {
        if v := os.Getenv(key); v != "" {
                return v
        }
        return fallback
}

func getEnvInt(key string, fallback int) int {
        if v := os.Getenv(key); v != "" {
                if n, err := strconv.Atoi(v); err == nil {
                        return n
                }
        }
        return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
        if v := os.Getenv(key); v != "" {
                if f, err := strconv.ParseFloat(v, 64); err == nil {
                        return f
                }
        }
        return fallback
}

func getEnvBool(key string, fallback bool) bool {
        if v := os.Getenv(key); v != "" {
                if b, err := strconv.ParseBool(v); err == nil {
                        return b
                }
        }
        return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
        if v := os.Getenv(key); v != "" {
                if d, err := time.ParseDuration(v); err == nil {
                        return d
                }
        }
        return fallback
}

func getEnvSlice(key string, fallback []string) []string {
        if v := os.Getenv(key); v != "" {
                parts := strings.Split(v, ",")
                for i, p := range parts {
                        parts[i] = strings.TrimSpace(p)
                }
                return parts
        }
        return fallback
}
