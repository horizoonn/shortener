package links

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"

	core_errors "github.com/horizoonn/shortener/internal/errors"
)

const (
	DefaultCodeAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	DefaultCodeLength   = 8
)

type CodeGenerator struct {
	alphabet string
	length   int
	reader   io.Reader
}

func NewCodeGenerator(alphabet string, length int, reader io.Reader) (*CodeGenerator, error) {
	if alphabet == "" {
		return nil, fmt.Errorf("code alphabet is empty: %w", core_errors.ErrInvalidArgument)
	}
	if length <= 0 {
		return nil, fmt.Errorf("code length must be positive: %w", core_errors.ErrInvalidArgument)
	}
	if reader == nil {
		return nil, fmt.Errorf("random reader is nil: %w", core_errors.ErrInvalidArgument)
	}

	seen := make(map[rune]struct{}, len(alphabet))
	for _, char := range alphabet {
		if _, ok := seen[char]; ok {
			return nil, fmt.Errorf("code alphabet contains duplicate character %q: %w", char, core_errors.ErrInvalidArgument)
		}
		seen[char] = struct{}{}
	}

	return &CodeGenerator{
		alphabet: alphabet,
		length:   length,
		reader:   reader,
	}, nil
}

func NewDefaultCodeGenerator() (*CodeGenerator, error) {
	return NewCodeGenerator(DefaultCodeAlphabet, DefaultCodeLength, rand.Reader)
}

func (g *CodeGenerator) Generate() (string, error) {
	if g == nil {
		return "", fmt.Errorf("code generator is nil: %w", core_errors.ErrInvalidArgument)
	}

	max := big.NewInt(int64(len(g.alphabet)))
	code := make([]byte, g.length)

	for i := range code {
		n, err := rand.Int(g.reader, max)
		if err != nil {
			return "", fmt.Errorf("generate random code character: %w", err)
		}
		code[i] = g.alphabet[n.Int64()]
	}

	return string(code), nil
}
