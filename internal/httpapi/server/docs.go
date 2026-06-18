package server

import (
	"net/http"

	"github.com/horizoonn/shortener/internal/docs"
)

func DocsRoute() Route {
	return Route{
		Method: http.MethodGet,
		Path:   "/docs",
		Handler: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(docs.SwaggerUIHTML)
		},
	}
}

func DocsOpenAPIRoute() Route {
	return Route{
		Method: http.MethodGet,
		Path:   "/docs/openapi.yaml",
		Handler: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(docs.OpenAPISpec)
		},
	}
}
