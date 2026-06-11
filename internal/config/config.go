package config

import (
	"fmt"
	"time"
	_ "time/tzdata"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Environment string `envconfig:"ENVIRONMENT" default:"development"`
	RawTimeZone string `envconfig:"TIME_ZONE" default:"UTC"`
	TimeZone    *time.Location

	Logger LoggerConfig
	HTTP   HTTPConfig

	DatabaseURL string `envconfig:"DATABASE_URL" default:"postgres://shortener:shortener@localhost:5432/shortener?sslmode=disable"`

	RedisAddr     string        `envconfig:"REDIS_ADDR" default:"localhost:6379"`
	RedisPassword string        `envconfig:"REDIS_PASSWORD" default:""`
	RedisDB       int           `envconfig:"REDIS_DB" default:"0"`
	RedisCacheTTL time.Duration `envconfig:"REDIS_CACHE_TTL" default:"10m"`
}

type LoggerConfig struct {
	Level  string `envconfig:"LOG_LEVEL" default:"DEBUG"`
	Folder string `envconfig:"LOG_FOLDER" default:".out/logs"`
}

type HTTPConfig struct {
	Addr              string        `envconfig:"HTTP_ADDR" default:":8080"`
	ShutdownTimeout   time.Duration `envconfig:"HTTP_SHUTDOWN_TIMEOUT" default:"10s"`
	ReadHeaderTimeout time.Duration `envconfig:"HTTP_READ_HEADER_TIMEOUT" default:"5s"`
	ReadTimeout       time.Duration `envconfig:"HTTP_READ_TIMEOUT" default:"10s"`
	WriteTimeout      time.Duration `envconfig:"HTTP_WRITE_TIMEOUT" default:"10s"`
	IdleTimeout       time.Duration `envconfig:"HTTP_IDLE_TIMEOUT" default:"60s"`
	AllowedOrigins    []string      `envconfig:"HTTP_ALLOWED_ORIGINS" default:"*"`
	AllowedMethods    []string      `envconfig:"HTTP_ALLOWED_METHODS" default:"GET,POST,OPTIONS"`
}

func Load() (Config, error) {
	var cfg Config
	if err := envconfig.Process("SHORTENER", &cfg); err != nil {
		return Config{}, fmt.Errorf("process env config: %w", err)
	}

	zone, err := time.LoadLocation(cfg.RawTimeZone)
	if err != nil {
		return Config{}, fmt.Errorf("load time zone %q: %w", cfg.RawTimeZone, err)
	}
	cfg.TimeZone = zone

	if err := validate(cfg); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

func validate(cfg Config) error {
	if cfg.Logger.Level == "" {
		return fmt.Errorf("logger level is required")
	}
	if cfg.Logger.Folder == "" {
		return fmt.Errorf("logger folder is required")
	}
	if cfg.HTTP.Addr == "" {
		return fmt.Errorf("http addr is required")
	}
	if cfg.HTTP.ShutdownTimeout <= 0 {
		return fmt.Errorf("http shutdown timeout must be positive")
	}
	if cfg.HTTP.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("http read header timeout must be positive")
	}
	if cfg.HTTP.ReadTimeout <= 0 {
		return fmt.Errorf("http read timeout must be positive")
	}
	if cfg.HTTP.WriteTimeout <= 0 {
		return fmt.Errorf("http write timeout must be positive")
	}
	if cfg.HTTP.IdleTimeout <= 0 {
		return fmt.Errorf("http idle timeout must be positive")
	}
	if len(cfg.HTTP.AllowedOrigins) == 0 {
		return fmt.Errorf("http allowed origins is required")
	}
	if len(cfg.HTTP.AllowedMethods) == 0 {
		return fmt.Errorf("http allowed methods is required")
	}
	return nil
}
