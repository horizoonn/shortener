package http

import (
	"context"
	nethttp "net/http"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/httpapi/request"
	"github.com/horizoonn/shortener/internal/httpapi/response"
	"github.com/horizoonn/shortener/internal/logger"
	"go.uber.org/zap"
)

const linkCodePathValue = "code"

func (h *Handler) RedirectLink(w nethttp.ResponseWriter, r *nethttp.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	responseHandler := response.NewHTTPResponseHandler(log, w)

	code, err := request.GetStringPathValue(r, linkCodePathValue)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to resolve short link")
		return
	}

	link, err := h.linksService.ResolveLink(ctx, code)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to resolve short link")
		return
	}

	if err := h.recordClick(ctx, r, link.ID); err != nil {
		log.Warn("failed to record short link click", zap.Error(err), zap.String("code", code))
	}

	nethttp.Redirect(w, r, link.OriginalURL, nethttp.StatusFound)
}

func (h *Handler) recordClick(ctx context.Context, r *nethttp.Request, linkID uuid.UUID) error {
	if h.clickRecorder == nil {
		return nil
	}

	var ip string
	if h.ipResolver != nil {
		ip = h.ipResolver.Resolve(r)
	} else {
		ip = request.ClientIP(r)
	}

	var ipPtr *string
	if ip != "" {
		ipPtr = &ip
	}

	return h.clickRecorder.RecordClick(
		ctx,
		linkID,
		r.UserAgent(),
		optionalHeader(r, "Referer"),
		ipPtr,
	)
}

func optionalHeader(r *nethttp.Request, name string) *string {
	value := r.Header.Get(name)
	if value == "" {
		return nil
	}

	return &value
}
