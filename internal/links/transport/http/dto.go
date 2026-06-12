package http

import (
	"time"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/links"
)

type CreateLinkRequest struct {
	OriginalURL string  `json:"original_url"`
	CustomAlias *string `json:"custom_alias"`
}

type CreateLinkResponse struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	OriginalURL string    `json:"original_url"`
	ShortURL    string    `json:"short_url"`
	IsCustom    bool      `json:"is_custom"`
	CreatedAt   time.Time `json:"created_at"`
}

func createLinkResponseFromDomain(link links.Link, shortURL string) CreateLinkResponse {
	return CreateLinkResponse{
		ID:          link.ID,
		Code:        link.Code,
		OriginalURL: link.OriginalURL,
		ShortURL:    shortURL,
		IsCustom:    link.IsCustom,
		CreatedAt:   link.CreatedAt,
	}
}
