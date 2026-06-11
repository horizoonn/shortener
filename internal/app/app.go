package app

import (
	"context"
	"fmt"

	"github.com/horizoonn/shortener/internal/config"
	"github.com/horizoonn/shortener/internal/httpapi/middleware"
	"github.com/horizoonn/shortener/internal/httpapi/server"
	"github.com/horizoonn/shortener/internal/logger"
)

type App struct {
	httpServer *server.HTTPServer
}

func New(cfg config.Config, log *logger.Logger) (*App, error) {
	if log == nil {
		return nil, fmt.Errorf("logger is required")
	}

	httpServer, err := server.NewHTTPServer(
		cfg.HTTP,
		log,
		middleware.RequestID(),
		middleware.Logger(log),
		middleware.Trace(),
		middleware.Panic(),
		middleware.CORS(cfg.HTTP.AllowedOrigins, cfg.HTTP.AllowedMethods),
	)
	if err != nil {
		return nil, fmt.Errorf("init HTTP server: %w", err)
	}
	httpServer.RegisterRoutes(server.HealthRoute())

	return &App{
		httpServer: httpServer,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	return a.httpServer.Run(ctx)
}
