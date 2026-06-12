package http

import (
	"context"
	nethttp "net/http"
	"strings"

	"github.com/horizoonn/shortener/internal/httpapi/server"
	"github.com/horizoonn/shortener/internal/links"
)

type Handler struct {
	linksService  LinksService
	publicBaseURL string
}

type LinksService interface {
	CreateLink(ctx context.Context, originalURL string, customAlias *string) (links.Link, error)
}

func NewHandler(linksService LinksService, publicBaseURL string) *Handler {
	return &Handler{
		linksService:  linksService,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
	}
}

func (h *Handler) Routes() []server.Route {
	return []server.Route{
		{
			Method:  nethttp.MethodPost,
			Path:    "/shorten",
			Handler: h.CreateLink,
		},
	}
}

func (h *Handler) shortURL(code string) string {
	return h.publicBaseURL + "/s/" + code
}
