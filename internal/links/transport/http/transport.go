package http

import (
	"context"
	nethttp "net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/analytics"
	"github.com/horizoonn/shortener/internal/httpapi/server"
	"github.com/horizoonn/shortener/internal/links"
)

type Handler struct {
	linksService    LinksService
	clickRecorder   ClickRecorder
	analyticsReader AnalyticsReader
	publicBaseURL   string
}

type LinksService interface {
	CreateLink(ctx context.Context, originalURL string, customAlias *string) (links.Link, error)
	ResolveLink(ctx context.Context, code string) (links.Link, error)
}

type ClickRecorder interface {
	RecordClick(ctx context.Context, linkID uuid.UUID, userAgent string, referer *string, ip *string) error
}

type AnalyticsReader interface {
	GetLinkAnalytics(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter, recentLimit int) (analytics.LinkAnalytics, error)
}

func NewHandler(linksService LinksService, publicBaseURL string) *Handler {
	return NewHandlerWithDependencies(linksService, nil, nil, publicBaseURL)
}

func NewHandlerWithClickRecorder(
	linksService LinksService,
	clickRecorder ClickRecorder,
	publicBaseURL string,
) *Handler {
	return NewHandlerWithDependencies(linksService, clickRecorder, nil, publicBaseURL)
}

func NewHandlerWithDependencies(
	linksService LinksService,
	clickRecorder ClickRecorder,
	analyticsReader AnalyticsReader,
	publicBaseURL string,
) *Handler {
	return &Handler{
		linksService:    linksService,
		clickRecorder:   clickRecorder,
		analyticsReader: analyticsReader,
		publicBaseURL:   strings.TrimRight(publicBaseURL, "/"),
	}
}

func (h *Handler) Routes() []server.Route {
	return []server.Route{
		{
			Method:  nethttp.MethodPost,
			Path:    "/shorten",
			Handler: h.CreateLink,
		},
		{
			Method:  nethttp.MethodGet,
			Path:    "/analytics/{code}",
			Handler: h.GetAnalytics,
		},
	}
}

func (h *Handler) RedirectRoutes() []server.Route {
	return []server.Route{
		{
			Method:  nethttp.MethodGet,
			Path:    "/s/{code}",
			Handler: h.RedirectLink,
		},
	}
}

func (h *Handler) shortURL(code string) string {
	return h.publicBaseURL + "/s/" + code
}
