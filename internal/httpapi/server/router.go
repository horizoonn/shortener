package server

import (
	"net/http"

	"github.com/horizoonn/shortener/internal/httpapi/middleware"
)

type APIVersion string

const (
	APIVersion1 = APIVersion("v1")
)

type APIVersionRouter struct {
	apiVersion APIVersion
	routes     []Route
	middleware []middleware.Middleware
}

func NewAPIVersionRouter(apiVersion APIVersion, middleware ...middleware.Middleware) *APIVersionRouter {
	return &APIVersionRouter{
		apiVersion: apiVersion,
		middleware: middleware,
	}
}

func (r *APIVersionRouter) AddRoutes(routes ...Route) {
	r.routes = append(r.routes, routes...)
}

func (r *APIVersionRouter) RegisterRoutesTo(mux *http.ServeMux) {
	for _, route := range r.routes {
		pattern := route.Method + " /api/" + string(r.apiVersion) + route.Path
		handler := middleware.Chain(route.WithMiddleware(), r.middleware...)

		mux.Handle(pattern, handler)
	}
}
