package middleware

import (
	"net/http"

	"github.com/horizoonn/shortener/internal/httpapi/response"
	"github.com/horizoonn/shortener/internal/logger"
)

func Panic() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.FromContext(r.Context())
			rw := response.NewResponseWriter(w)
			responseHandler := response.NewHTTPResponseHandler(log, rw)

			defer func() {
				if recovered := recover(); recovered != nil {
					responseHandler.PanicResponse(recovered, "recovered from unexpected panic during request processing")
				}
			}()

			next.ServeHTTP(rw, r)
		})
	}
}
