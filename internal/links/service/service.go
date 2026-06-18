package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/links"
)

const MaxCodeGenerationAttempts = 5

type Metrics interface {
	RecordLinkCreated(isCustom bool)
	RecordLinkResolved()
}

type Service struct {
	linksRepository LinksRepository
	codeGenerator   CodeGenerator
	linkCache       LinkCache
	metrics         Metrics
}

type LinksRepository interface {
	CreateLink(ctx context.Context, link links.Link) (links.Link, error)
	GetLinkByCode(ctx context.Context, code string) (links.Link, error)
	DisableLink(ctx context.Context, code string) (links.Link, error)
}

type CodeGenerator interface {
	Generate() (string, error)
}

type LinkCache interface {
	GetLink(ctx context.Context, code string) (links.Link, error)
	SetLink(ctx context.Context, link links.Link) error
	SetLinkNotFound(ctx context.Context, code string) error
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

func (s *Service) WithMetrics(metrics Metrics) *Service {
	if s == nil {
		return nil
	}
	s.metrics = metrics
	return s
}

func newLink(code string, originalURL string, isCustom bool, expiresAt *time.Time) links.Link {
	return links.Link{
		ID:          uuid.New(),
		Code:        code,
		OriginalURL: originalURL,
		IsCustom:    isCustom,
		ExpiresAt:   expiresAt,
	}
}
