package http

import (
	"fmt"
	nethttp "net/http"

	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/httpapi/request"
	"github.com/horizoonn/shortener/internal/httpapi/response"
	"github.com/horizoonn/shortener/internal/logger"
	qr_generator "github.com/horizoonn/shortener/internal/qr"
)

const qrCacheControl = "public, max-age=3600"

func (h *Handler) GetQRCode(w nethttp.ResponseWriter, r *nethttp.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	responseHandler := response.NewHTTPResponseHandler(log, w)

	code, err := request.GetStringPathValue(r, linkCodePathValue)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get short link QR code")
		return
	}
	if h.linkResolver == nil {
		responseHandler.ErrorResponse(
			fmt.Errorf("link resolver is nil: %w", core_errors.ErrInternal),
			"failed to get short link QR code",
		)
		return
	}
	if h.qrGenerator == nil {
		responseHandler.ErrorResponse(
			fmt.Errorf("qr generator is nil: %w", core_errors.ErrInternal),
			"failed to get short link QR code",
		)
		return
	}

	size, err := parseQRSize(r)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to parse QR size")
		return
	}

	link, err := h.linkResolver.ResolveLink(ctx, code)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to resolve short link for QR code")
		return
	}

	png, err := h.qrGenerator.GeneratePNG(h.shortURL(link.Code), size)
	if err != nil {
		responseHandler.ErrorResponse(fmt.Errorf("generate QR code: %w", err), "failed to generate short link QR code")
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", qrCacheControl)
	w.WriteHeader(nethttp.StatusOK)
	// #nosec G705 -- writing generated binary PNG image
	_, _ = w.Write(png)
}

func parseQRSize(r *nethttp.Request) (int, error) {
	size, err := request.GetIntQueryParam(r, "size")
	if err != nil {
		return 0, err
	}
	if size == nil {
		return qr_generator.DefaultSize, nil
	}
	if err := qr_generator.ValidateSize(*size); err != nil {
		return 0, err
	}

	return *size, nil
}
