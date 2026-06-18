package http

import (
	"fmt"
	nethttp "net/http"
	"strconv"
	"time"

	"github.com/horizoonn/shortener/internal/analytics"
	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/httpapi/response"
	"github.com/horizoonn/shortener/internal/logger"
)

const (
	analyticsDateLayout     = "2006-01-02"
	defaultRecentClickLimit = 20
	maxRecentClickLimit     = 100
)

func (h *Handler) GetAnalytics(w nethttp.ResponseWriter, r *nethttp.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	responseHandler := response.NewHTTPResponseHandler(log, w)

	code := r.PathValue(linkCodePathValue)
	if code == "" {
		responseHandler.ErrorResponse(
			fmt.Errorf("link code is empty: %w", core_errors.ErrInvalidArgument),
			"failed to get short link analytics",
		)
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
	query := r.URL.Query()

	from, err := parseOptionalDate(query.Get("from"), "from")
	if err != nil {
		return analytics.ClickFilter{}, 0, err
	}

	to, err := parseOptionalDate(query.Get("to"), "to")
	if err != nil {
		return analytics.ClickFilter{}, 0, err
	}
	if to != nil {
		inclusiveTo := to.AddDate(0, 0, 1)
		to = &inclusiveTo
	}

	recentLimit, err := parseRecentLimit(query.Get("recent_limit"))
	if err != nil {
		return analytics.ClickFilter{}, 0, err
	}

	return analytics.ClickFilter{
		From: from,
		To:   to,
	}, recentLimit, nil
}

func parseOptionalDate(value string, fieldName string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse(analyticsDateLayout, value)
	if err != nil {
		return nil, fmt.Errorf("%s must use YYYY-MM-DD format: %w", fieldName, core_errors.ErrInvalidArgument)
	}

	return &parsed, nil
}

func parseRecentLimit(value string) (int, error) {
	if value == "" {
		return defaultRecentClickLimit, nil
	}

	limit, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("recent_limit must be a number: %w", core_errors.ErrInvalidArgument)
	}
	if limit <= 0 {
		return 0, fmt.Errorf("recent_limit must be positive: %w", core_errors.ErrInvalidArgument)
	}
	if limit > maxRecentClickLimit {
		return 0, fmt.Errorf("recent_limit must be less than or equal to %d: %w", maxRecentClickLimit, core_errors.ErrInvalidArgument)
	}

	return limit, nil
}
