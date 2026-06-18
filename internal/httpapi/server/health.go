package server

import (
	"net/http"

	"github.com/horizoonn/shortener/internal/httpapi/response"
)

func HealthRoute() Route {
	return Route{
		Method: "GET",
		Path:   "/healthz",
		Handler: func(w http.ResponseWriter, _ *http.Request) {
			response.WriteJSON(w, http.StatusOK, map[string]string{
				"status": "ok",
			})
		},
	}
}
