package links

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"

	core_errors "github.com/horizoonn/shortener/internal/errors"
)

const (
	CustomAliasMinLength = 3
	CustomAliasMaxLength = 64
)

var reservedCustomAliases = map[string]struct{}{
	"healthz": {},
	"readyz":  {},
	"api":     {},
	"swagger": {},
	"static":  {},
	"assets":  {},
}

func ValidateOriginalURL(rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("original URL is empty: %w", core_errors.ErrInvalidArgument)
	}
	if strings.TrimSpace(rawURL) != rawURL {
		return "", fmt.Errorf("original URL contains leading or trailing whitespace: %w", core_errors.ErrInvalidArgument)
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse original URL: %w", core_errors.ErrInvalidArgument)
	}

	switch parsedURL.Scheme {
	case "http", "https":
	default:
		return "", fmt.Errorf("unsupported original URL scheme %q: %w", parsedURL.Scheme, core_errors.ErrInvalidArgument)
	}

	if parsedURL.Host == "" {
		return "", fmt.Errorf("original URL host is empty: %w", core_errors.ErrInvalidArgument)
	}

	return rawURL, nil
}

func ValidateCustomAlias(alias string) error {
	if alias == "" {
		return fmt.Errorf("custom alias is empty: %w", core_errors.ErrInvalidArgument)
	}
	if len(alias) < CustomAliasMinLength {
		return fmt.Errorf("custom alias is shorter than %d characters: %w", CustomAliasMinLength, core_errors.ErrInvalidArgument)
	}
	if len(alias) > CustomAliasMaxLength {
		return fmt.Errorf("custom alias is longer than %d characters: %w", CustomAliasMaxLength, core_errors.ErrInvalidArgument)
	}
	if _, ok := reservedCustomAliases[strings.ToLower(alias)]; ok {
		return fmt.Errorf("custom alias %q is reserved: %w", alias, core_errors.ErrInvalidArgument)
	}

	for _, char := range alias {
		if isAllowedAliasChar(char) {
			continue
		}

		return fmt.Errorf("custom alias contains invalid character %q: %w", char, core_errors.ErrInvalidArgument)
	}

	return nil
}

func isAllowedAliasChar(char rune) bool {
	if char == '-' || char == '_' {
		return true
	}
	if char > unicode.MaxASCII {
		return false
	}

	return unicode.IsLetter(char) || unicode.IsDigit(char)
}
