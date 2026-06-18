package qr

import (
	"fmt"

	core_errors "github.com/horizoonn/shortener/internal/errors"
	qrcode "github.com/skip2/go-qrcode"
)

const (
	DefaultSize = 256
	MinSize     = 128
	MaxSize     = 1024
)

type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) GeneratePNG(content string, size int) ([]byte, error) {
	if err := ValidateSize(size); err != nil {
		return nil, err
	}
	if content == "" {
		return nil, fmt.Errorf("qr content is empty: %w", core_errors.ErrInvalidArgument)
	}

	png, err := qrcode.Encode(content, qrcode.Medium, size)
	if err != nil {
		return nil, fmt.Errorf("encode QR PNG: %w", err)
	}

	return png, nil
}

func ValidateSize(size int) error {
	if size < MinSize {
		return fmt.Errorf("qr size must be at least %d: %w", MinSize, core_errors.ErrInvalidArgument)
	}
	if size > MaxSize {
		return fmt.Errorf("qr size must be at most %d: %w", MaxSize, core_errors.ErrInvalidArgument)
	}

	return nil
}
