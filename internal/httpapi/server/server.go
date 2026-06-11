package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/horizoonn/shortener/internal/httpapi/middleware"
	"github.com/horizoonn/shortener/internal/logger"
	"go.uber.org/zap"
)

type HTTPServer struct {
	mux        *http.ServeMux
	config     Config
	log        *logger.Logger
	middleware []middleware.Middleware
}

func NewHTTPServer(config Config, log *logger.Logger, middleware ...middleware.Middleware) (*HTTPServer, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("validate HTTP server config: %w", err)
	}
	if log == nil {
		log = &logger.Logger{Logger: zap.NewNop()}
	}

	return &HTTPServer{
		mux:        http.NewServeMux(),
		config:     config,
		log:        log,
		middleware: middleware,
	}, nil
}

func (s *HTTPServer) RegisterAPIRouters(routers ...*APIVersionRouter) {
	for _, router := range routers {
		router.RegisterRoutesTo(s.mux)
	}
}

func (s *HTTPServer) RegisterRoutes(routes ...Route) {
	for _, route := range routes {
		pattern := route.Method + " " + route.Path
		handler := route.WithMiddleware()

		s.mux.Handle(pattern, handler)
	}
}

func (s *HTTPServer) Handler() http.Handler {
	return middleware.Chain(s.mux, s.middleware...)
}

func (s *HTTPServer) Run(ctx context.Context) error {
	server := &http.Server{
		Addr:              s.config.Addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: s.config.ReadHeaderTimeout,
		ReadTimeout:       s.config.ReadTimeout,
		WriteTimeout:      s.config.WriteTimeout,
		IdleTimeout:       s.config.IdleTimeout,
	}

	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		s.log.Info("start HTTP server", zap.String("addr", s.config.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("listen and serve HTTP: %w", err)
		}
	case <-ctx.Done():
		s.log.Info("shutdown HTTP server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}

		s.log.Info("HTTP server stopped")
	}

	return nil
}
