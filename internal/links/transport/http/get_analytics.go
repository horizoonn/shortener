package http

import (
	"fmt"
	nethttp "net/http"

	"github.com/horizoonn/shortener/internal/analytics"
	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/httpapi/request"
	"github.com/horizoonn/shortener/internal/httpapi/response"
	"github.com/horizoonn/shortener/internal/logger"
)

const (
	defaultRecentClickLimit = 20
	maxRecentClickLimit     = 100
)

func (h *Handler) GetAnalytics(w nethttp.ResponseWriter, r *nethttp.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	responseHandler := response.NewHTTPResponseHandler(log, w)

	code, err := request.GetStringPathValue(r, linkCodePathValue)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get short link analytics")
		return
	}
	if h.analyticsReader == nil {
		responseHandler.ErrorResponse(
			fmt.Errorf("analytics reader is nil: %w", core_errors.ErrInternal),
			"failed to get short link analytics",
		)
		return
	}

	filter, recentLimit, err := parseAnalyticsQuery(r)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to parse analytics query")
		return
	}

	link, err := h.linksService.ResolveLink(ctx, code)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to resolve short link for analytics")
		return
	}

	linkAnalytics, err := h.analyticsReader.GetLinkAnalytics(ctx, link.ID, filter, recentLimit)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get short link analytics")
		return
	}

	responseHandler.JSONResponse(analyticsResponseFromDomain(link, linkAnalytics), nethttp.StatusOK)
}

func parseAnalyticsQuery(r *nethttp.Request) (analytics.ClickFilter, int, error) {
	from, err := request.GetDateQueryParam(r, "from")
	if err != nil {
		return analytics.ClickFilter{}, 0, err
	}

	to, err := request.GetDateQueryParam(r, "to")
	if err != nil {
		return analytics.ClickFilter{}, 0, err
	}
	if to != nil {
		inclusiveTo := to.AddDate(0, 0, 1)
		to = &inclusiveTo
	}

	recentLimit, err := parseRecentLimit(r)
	if err != nil {
		return analytics.ClickFilter{}, 0, err
	}

	return analytics.ClickFilter{
		From: from,
		To:   to,
	}, recentLimit, nil
}

func parseRecentLimit(r *nethttp.Request) (int, error) {
	limit, err := request.GetIntQueryParam(r, "recent_limit")
	if err != nil {
		return 0, err
	}
	if limit == nil {
		return defaultRecentClickLimit, nil
	}
	if *limit <= 0 {
		return 0, fmt.Errorf("recent_limit must be positive: %w", core_errors.ErrInvalidArgument)
	}
	if *limit > maxRecentClickLimit {
		return 0, fmt.Errorf("recent_limit must be less than or equal to %d: %w", maxRecentClickLimit, core_errors.ErrInvalidArgument)
	}

	return *limit, nil
}
