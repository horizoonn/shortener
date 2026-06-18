package qr

import (
	"bytes"
	"errors"
	"testing"

	core_errors "github.com/horizoonn/shortener/internal/errors"
)

func TestGeneratorGeneratePNG(t *testing.T) {
	t.Parallel()

	generator := NewGenerator()

	png, err := generator.GeneratePNG("http://localhost:8080/s/abc12345", DefaultSize)
	if err != nil {
		t.Fatalf("generate PNG: %v", err)
	}
	if len(png) == 0 {
		t.Fatal("expected PNG bytes")
	}
	if !bytes.HasPrefix(png, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		t.Fatal("expected PNG signature")
	}
}

func TestGeneratorGeneratePNGInvalidSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size int
	}{
		{name: "too small", size: MinSize - 1},
		{name: "too large", size: MaxSize + 1},
	}

	generator := NewGenerator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := generator.GeneratePNG("http://localhost:8080/s/abc12345", tt.size)
			if !errors.Is(err, core_errors.ErrInvalidArgument) {
				t.Fatalf("expected invalid argument, got %v", err)
			}
		})
	}
}
