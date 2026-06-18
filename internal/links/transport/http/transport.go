package http

import (
	"context"
	nethttp "net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/analytics"
	"github.com/horizoonn/shortener/internal/httpapi/request"
	"github.com/horizoonn/shortener/internal/httpapi/server"
	"github.com/horizoonn/shortener/internal/links"
	qr_generator "github.com/horizoonn/shortener/internal/qr"
)

type Handler struct {
	linksService    LinksService
	linkResolver    LinkResolver
	clickRecorder   ClickRecorder
	analyticsReader AnalyticsReader
	qrGenerator     QRGenerator
	ipResolver      *request.IPResolver
	publicBaseURL   string
}

type LinksService interface {
	CreateLink(ctx context.Context, originalURL string, customAlias *string, expiresAt *time.Time) (links.Link, error)
	GetLink(ctx context.Context, code string) (links.Link, error)
	ResolveLink(ctx context.Context, code string) (links.Link, error)
	DisableLink(ctx context.Context, code string) (links.Link, error)
}

type LinkResolver interface {
	ResolveLink(ctx context.Context, code string) (links.Link, error)
}

type ClickRecorder interface {
	RecordClick(ctx context.Context, linkID uuid.UUID, userAgent string, referer *string, ip *string) error
}

type AnalyticsReader interface {
	GetLinkAnalytics(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter, recentLimit int) (analytics.LinkAnalytics, error)
}

type QRGenerator interface {
	GeneratePNG(content string, size int) ([]byte, error)
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
	qrGenerators ...QRGenerator,
) *Handler {
	qrGenerator := QRGenerator(qr_generator.NewGenerator())
	if len(qrGenerators) > 0 && qrGenerators[0] != nil {
		qrGenerator = qrGenerators[0]
	}

	return &Handler{
		linksService:    linksService,
		linkResolver:    linksService,
		clickRecorder:   clickRecorder,
		analyticsReader: analyticsReader,
		qrGenerator:     qrGenerator,
		publicBaseURL:   strings.TrimRight(publicBaseURL, "/"),
	}
}

func (h *Handler) WithIPResolver(ipResolver *request.IPResolver) *Handler {
	if h == nil {
		return nil
	}
	h.ipResolver = ipResolver
	return h
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
		{
			Method:  nethttp.MethodDelete,
			Path:    "/links/{code}",
			Handler: h.DisableLink,
		},
		{
			Method:  nethttp.MethodGet,
			Path:    "/links/{code}/qr",
			Handler: h.GetQRCode,
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
