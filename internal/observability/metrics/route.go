package metrics

import (
	"net/http"

	"github.com/horizoonn/shortener/internal/httpapi/server"
)

func Route(handler http.Handler) server.Route {
	return server.Route{
		Method:  http.MethodGet,
		Path:    metricsPath,
		Handler: handler.ServeHTTP,
	}
}
