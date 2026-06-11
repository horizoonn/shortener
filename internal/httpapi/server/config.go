package server

import (
	"fmt"

	"github.com/horizoonn/shortener/internal/config"
)

type Config = config.HTTPConfig

func validateConfig(cfg Config) error {
	if cfg.Addr == "" {
		return fmt.Errorf("addr is empty")
	}
	if cfg.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be positive")
	}
	if cfg.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("read header timeout must be positive")
	}
	if cfg.ReadTimeout <= 0 {
		return fmt.Errorf("read timeout must be positive")
	}
	if cfg.WriteTimeout <= 0 {
		return fmt.Errorf("write timeout must be positive")
	}
	if cfg.IdleTimeout <= 0 {
		return fmt.Errorf("idle timeout must be positive")
	}
	if len(cfg.AllowedOrigins) == 0 {
		return fmt.Errorf("allowed origins is empty")
	}
	if len(cfg.AllowedMethods) == 0 {
		return fmt.Errorf("allowed methods is empty")
	}
	return nil
}
