package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/links"
)

const MaxCodeGenerationAttempts = 5

type Service struct {
	linksRepository LinksRepository
	codeGenerator   CodeGenerator
	linkCache       LinkCache
}

type LinksRepository interface {
	CreateLink(ctx context.Context, link links.Link) (links.Link, error)
	GetLinkByCode(ctx context.Context, code string) (links.Link, error)
}

type CodeGenerator interface {
	Generate() (string, error)
}

type LinkCache interface {
	GetLink(ctx context.Context, code string) (links.Link, error)
	SetLink(ctx context.Context, link links.Link) error
	DeleteLink(ctx context.Context, code string) error
}

func NewService(linksRepository LinksRepository, codeGenerator CodeGenerator) *Service {
	return NewServiceWithCache(linksRepository, codeGenerator, nil)
}

func NewServiceWithCache(
	linksRepository LinksRepository,
	codeGenerator CodeGenerator,
	linkCache LinkCache,
) *Service {
	return &Service{
		linksRepository: linksRepository,
		codeGenerator:   codeGenerator,
		linkCache:       linkCache,
	}
}

func newLink(code string, originalURL string, isCustom bool) links.Link {
	return links.Link{
		ID:          uuid.New(),
		Code:        code,
		OriginalURL: originalURL,
		IsCustom:    isCustom,
	}
}
