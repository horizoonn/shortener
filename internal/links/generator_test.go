package links

import (
	"errors"
	"io"
	"strings"
	"testing"

	core_errors "github.com/horizoonn/shortener/internal/errors"
)

func TestCodeGeneratorGenerate(t *testing.T) {
	t.Parallel()

	generator, err := NewCodeGenerator("abc123", 16, strings.NewReader(strings.Repeat("\x00", 16)))
	if err != nil {
		t.Fatalf("new code generator: %v", err)
	}

	code, err := generator.Generate()
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}

	if len(code) != 16 {
		t.Fatalf("expected code length 16, got %d", len(code))
	}
	for _, char := range code {
		if !strings.ContainsRune("abc123", char) {
			t.Fatalf("generated code contains character outside alphabet: %q", char)
		}
	}
}

func TestNewCodeGeneratorInvalidConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		alphabet string
		length   int
		reader   io.Reader
	}{
		{
			name:     "empty alphabet",
			alphabet: "",
			length:   8,
			reader:   strings.NewReader("random"),
		},
		{
			name:     "duplicate alphabet characters",
			alphabet: "aabc",
			length:   8,
			reader:   strings.NewReader("random"),
		},
		{
			name:     "non ASCII alphabet",
			alphabet: "abcд",
			length:   8,
			reader:   strings.NewReader("random"),
		},
		{
			name:     "zero length",
			alphabet: "abc",
			length:   0,
			reader:   strings.NewReader("random"),
		},
		{
			name:     "nil reader",
			alphabet: "abc",
			length:   8,
			reader:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewCodeGenerator(tt.alphabet, tt.length, tt.reader)
			if !errors.Is(err, core_errors.ErrInvalidArgument) {
				t.Fatalf("expected invalid argument, got %v", err)
			}
		})
	}
}

func TestDefaultCodeGeneratorConfig(t *testing.T) {
	t.Parallel()

	generator, err := NewDefaultCodeGenerator()
	if err != nil {
		t.Fatalf("new default code generator: %v", err)
	}

	code, err := generator.Generate()
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}

	if len(code) != DefaultCodeLength {
		t.Fatalf("expected default code length %d, got %d", DefaultCodeLength, len(code))
	}
	for _, char := range code {
		if !strings.ContainsRune(DefaultCodeAlphabet, char) {
			t.Fatalf("generated code contains character outside default alphabet: %q", char)
		}
	}
}
