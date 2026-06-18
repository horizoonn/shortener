package http

import (
	"fmt"
	nethttp "net/http"

	"github.com/horizoonn/shortener/internal/httpapi/request"
	"github.com/horizoonn/shortener/internal/httpapi/response"
	"github.com/horizoonn/shortener/internal/logger"
)

const maxCreateLinkRequestBytes = 64 * 1024

func (h *Handler) CreateLink(w nethttp.ResponseWriter, r *nethttp.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	responseHandler := response.NewHTTPResponseHandler(log, w)

	var requestBody CreateLinkRequest
	if err := request.DecodeAndValidateJSON(w, r, &requestBody, maxCreateLinkRequestBytes); err != nil {
		responseHandler.ErrorResponse(
			fmt.Errorf("decode and validate create link request: %w", err),
			"failed to decode and validate create link HTTP request",
		)
		return
	}

	link, err := h.linksService.CreateLink(ctx, requestBody.OriginalURL, requestBody.CustomAlias)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to create short link")
		return
	}

	responseBody := createLinkResponseFromDomain(link, h.shortURL(link.Code))
	responseHandler.JSONResponse(responseBody, nethttp.StatusCreated)
}
