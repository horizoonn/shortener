package response

import (
	"errors"
	"net/http"

	core_errors "github.com/horizoonn/shortener/internal/errors"
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

func (h *HTTPResponseHandler) JSONResponse(responseBody any, statusCode int) {
	WriteJSON(h.w, statusCode, responseBody)
}

func (h *HTTPResponseHandler) ErrorResponse(err error, message string) {
	statusCode := http.StatusInternalServerError
	publicMessage := "internal server error"
	code := "internal_error"
	logFunc := h.log.Error

	switch {
	case errors.Is(err, core_errors.ErrInvalidArgument):
		statusCode = http.StatusBadRequest
		publicMessage = "invalid request"
		code = "invalid_argument"
		logFunc = h.log.Warn
	case errors.Is(err, core_errors.ErrNotFound):
		statusCode = http.StatusNotFound
		publicMessage = "resource not found"
		code = "not_found"
		logFunc = h.log.Debug
	case errors.Is(err, core_errors.ErrConflict):
		statusCode = http.StatusConflict
		publicMessage = "resource conflict"
		code = "conflict"
		logFunc = h.log.Warn
	}

	logFunc(message, zap.Error(err))
	WriteError(h.w, statusCode, publicMessage, code)
}

func (h *HTTPResponseHandler) PanicResponse(recovered any, message string) {
	h.log.Error(message, zap.Any("panic", recovered))

	if h.w.HeadersWritten() {
		return
	}

	WriteError(h.w, http.StatusInternalServerError, "internal server error", "internal_error")
}
