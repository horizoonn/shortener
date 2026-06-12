package http

import (
	nethttp "net/http"

	"github.com/horizoonn/shortener/internal/httpapi/request"
	"github.com/horizoonn/shortener/internal/httpapi/response"
	"github.com/horizoonn/shortener/internal/logger"
)

func (h *Handler) DisableLink(w nethttp.ResponseWriter, r *nethttp.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	responseHandler := response.NewHTTPResponseHandler(log, w)

	code, err := request.GetStringPathValue(r, linkCodePathValue)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to disable short link")
		return
	}

	if _, err := h.linksService.DisableLink(ctx, code); err != nil {
		responseHandler.ErrorResponse(err, "failed to disable short link")
		return
	}

	responseHandler.NoContentResponse()
}
