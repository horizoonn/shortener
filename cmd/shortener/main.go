package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/horizoonn/shortener/internal/app"
	"github.com/horizoonn/shortener/internal/config"
	"github.com/horizoonn/shortener/internal/logger"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("application failed: %v", err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	zapLogger, err := logger.New(cfg.Logger)
	if err != nil {
		return err
	}
	defer zapLogger.Close()

	zapLogger.Debug("application time zone", zap.String("zone", cfg.TimeZone.String()))

	application, err := app.New(ctx, cfg, zapLogger)
	if err != nil {
		return err
	}

	if err := application.Run(ctx); err != nil {
		return err
	}

	return nil
}
