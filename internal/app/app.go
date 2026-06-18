package app

import (
	"context"
	"fmt"
	"net/http"

	analytics_postgres "github.com/horizoonn/shortener/internal/analytics/postgres"
	"github.com/horizoonn/shortener/internal/config"
	"github.com/horizoonn/shortener/internal/httpapi/middleware"
	"github.com/horizoonn/shortener/internal/httpapi/response"
	"github.com/horizoonn/shortener/internal/httpapi/server"
	links_postgres "github.com/horizoonn/shortener/internal/links/postgres"
	"github.com/horizoonn/shortener/internal/logger"
	pgx_pool "github.com/horizoonn/shortener/internal/storage/postgres/pool/pgx"
)

type App struct {
	httpServer          *server.HTTPServer
	postgresPool        *pgx_pool.Pool
	linksRepository     *links_postgres.Repository
	analyticsRepository *analytics_postgres.Repository
}

func New(ctx context.Context, cfg config.Config, log *logger.Logger) (*App, error) {
	if log == nil {
		return nil, fmt.Errorf("logger is required")
	}

	log.Debug("initializing postgres connection pool")
	postgresPool, err := pgx_pool.NewPool(ctx, cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("init postgres pool: %w", err)
	}

	log.Debug("initializing links repository")
	linksRepository := links_postgres.NewRepository(postgresPool)

	log.Debug("initializing analytics repository")
	analyticsRepository := analytics_postgres.NewRepository(postgresPool)

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
		postgresPool.Close()
		return nil, fmt.Errorf("init HTTP server: %w", err)
	}
	httpServer.RegisterRoutes(server.HealthRoute(), readyRoute(postgresPool))

	return &App{
		httpServer:          httpServer,
		postgresPool:        postgresPool,
		linksRepository:     linksRepository,
		analyticsRepository: analyticsRepository,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	defer func() {
		a.postgresPool.Close()
	}()

	if err := a.httpServer.Run(ctx); err != nil {
		return err
	}

	return nil
}

func readyRoute(postgresPool *pgx_pool.Pool) server.Route {
	return server.Route{
		Method: "GET",
		Path:   "/readyz",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), postgresPool.OpTimeout())
			defer cancel()

			if err := postgresPool.Ping(ctx); err != nil {
				response.WriteError(w, http.StatusServiceUnavailable, "service is not ready", "not_ready")
				return
			}

			response.WriteJSON(w, http.StatusOK, map[string]string{
				"status": "ready",
			})
		},
	}
}
