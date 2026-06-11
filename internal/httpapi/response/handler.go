package response

import (
	"net/http"

	"github.com/horizoonn/shortener/internal/logger"
	"go.uber.org/zap"
)

type HTTPResponseHandler struct {
	log *logger.Logger
	w   *ResponseWriter
}

func NewHTTPResponseHandler(log *logger.Logger, w http.ResponseWriter) *HTTPResponseHandler {
	return &HTTPResponseHandler{
		log: log,
		w:   NewResponseWriter(w),
	}
}

func (h *HTTPResponseHandler) PanicResponse(recovered any, message string) {
	h.log.Error(message, zap.Any("panic", recovered))

	if h.w.HeadersWritten() {
		return
	}

	WriteError(h.w, http.StatusInternalServerError, "internal server error", "internal")
}
