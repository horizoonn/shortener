package app

import (
	"context"
	"fmt"
	"net/http"

	analytics_postgres "github.com/horizoonn/shortener/internal/analytics/postgres"
	analytics_service "github.com/horizoonn/shortener/internal/analytics/service"
	"github.com/horizoonn/shortener/internal/config"
	"github.com/horizoonn/shortener/internal/httpapi/middleware"
	"github.com/horizoonn/shortener/internal/httpapi/response"
	"github.com/horizoonn/shortener/internal/httpapi/server"
	"github.com/horizoonn/shortener/internal/links"
	links_postgres "github.com/horizoonn/shortener/internal/links/postgres"
	links_redis "github.com/horizoonn/shortener/internal/links/redis"
	links_service "github.com/horizoonn/shortener/internal/links/service"
	links_transport_http "github.com/horizoonn/shortener/internal/links/transport/http"
	"github.com/horizoonn/shortener/internal/logger"
	pgx_pool "github.com/horizoonn/shortener/internal/storage/postgres/pool/pgx"
	goredis "github.com/redis/go-redis/v9"
)

type App struct {
	httpServer          *server.HTTPServer
	postgresPool        *pgx_pool.Pool
	linksRepository     *links_postgres.Repository
	analyticsRepository *analytics_postgres.Repository
	linksCache          *links_redis.Cache
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

	log.Debug("initializing links redis cache")
	redisClient := goredis.NewClient(&goredis.Options{
		Addr:                  cfg.RedisAddr,
		Password:              cfg.RedisPassword,
		DB:                    cfg.RedisDB,
		DialTimeout:           cfg.RedisTimeout,
		ReadTimeout:           cfg.RedisTimeout,
		WriteTimeout:          cfg.RedisTimeout,
		ContextTimeoutEnabled: true,
	})
	linksCache, err := links_redis.NewCache(redisClient, cfg.RedisCacheTTL)
	if err != nil {
		_ = redisClient.Close()
		postgresPool.Close()
		return nil, fmt.Errorf("init links redis cache: %w", err)
	}

	log.Debug("initializing analytics repository")
	analyticsRepository := analytics_postgres.NewRepository(postgresPool)
	analyticsService := analytics_service.NewService(analyticsRepository)

	linksService := links_service.NewServiceWithCache(linksRepository, linksCodeGenerator, linksCache)
	linksHTTPHandler := links_transport_http.NewHandlerWithDependencies(
		linksService,
		analyticsService,
		analyticsService,
		cfg.HTTP.PublicBaseURL,
	)

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
		_ = linksCache.Close()
		postgresPool.Close()
		return nil, fmt.Errorf("init HTTP server: %w", err)
	}
	apiVersionRouterV1 := server.NewAPIVersionRouter(server.APIVersion1)
	apiVersionRouterV1.AddRoutes(linksHTTPHandler.Routes()...)

	httpServer.RegisterRoutes(server.HealthRoute(), readyRoute(postgresPool), staticUIRoute())
	httpServer.RegisterRoutes(linksHTTPHandler.RedirectRoutes()...)
	httpServer.RegisterAPIRouters(apiVersionRouterV1)

	return &App{
		httpServer:          httpServer,
		postgresPool:        postgresPool,
		linksRepository:     linksRepository,
		analyticsRepository: analyticsRepository,
		linksCache:          linksCache,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	defer func() {
		if a.linksCache != nil {
			_ = a.linksCache.Close()
		}
		a.postgresPool.Close()
	}()

	if err := a.httpServer.Run(ctx); err != nil {
		return err
	}

	return nil
}

func staticUIRoute() server.Route {
	fileServer := http.FileServer(http.Dir("web/public"))

	return server.Route{
		Method:  http.MethodGet,
		Path:    "/{$}",
		Handler: fileServer.ServeHTTP,
	}
}

func readyRoute(postgresPool *pgx_pool.Pool) server.Route {
	return server.Route{
		Method: http.MethodGet,
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
