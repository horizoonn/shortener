package middleware

import (
	"net/http"
	"strings"
)

func CORS(allowedOriginsList []string, allowedMethodsList []string) Middleware {
	allowedOrigins := make(map[string]struct{}, len(allowedOriginsList))
	for _, origin := range allowedOriginsList {
		allowedOrigins[origin] = struct{}{}
	}

	allowedMethods := strings.Join(allowedMethodsList, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := allowedOrigins["*"]; ok {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if _, ok := allowedOrigins[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			if allowedMethods != "" {
				w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
			}
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
