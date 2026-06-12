package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/horizoonn/shortener/internal/app"
	"github.com/horizoonn/shortener/internal/config"
	"github.com/horizoonn/shortener/internal/logger"
	"go.uber.org/zap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	time.Local = cfg.TimeZone

	zapLogger, err := logger.New(cfg.Logger)
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer zapLogger.Close()

	zapLogger.Debug("application time zone", zap.String("zone", time.Local.String()))

	application, err := app.New(ctx, cfg, zapLogger)
	if err != nil {
		zapLogger.Fatal("init app", zap.Error(err))
	}

	if err := application.Run(ctx); err != nil {
		zapLogger.Fatal("run app", zap.Error(err))
	}
}
