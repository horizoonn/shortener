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
	"github.com/horizoonn/shortener/internal/links"
	links_postgres "github.com/horizoonn/shortener/internal/links/postgres"
	links_service "github.com/horizoonn/shortener/internal/links/service"
	links_transport_http "github.com/horizoonn/shortener/internal/links/transport/http"
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
	linksCodeGenerator, err := links.NewDefaultCodeGenerator()
	if err != nil {
		postgresPool.Close()
		return nil, fmt.Errorf("init links code generator: %w", err)
	}
	linksService := links_service.NewService(linksRepository, linksCodeGenerator)
	linksHTTPHandler := links_transport_http.NewHandler(linksService, cfg.HTTP.PublicBaseURL)

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
	apiVersionRouterV1 := server.NewAPIVersionRouter(server.APIVersion1)
	apiVersionRouterV1.AddRoutes(linksHTTPHandler.Routes()...)

	httpServer.RegisterRoutes(server.HealthRoute(), readyRoute(postgresPool))
	httpServer.RegisterAPIRouters(apiVersionRouterV1)

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
