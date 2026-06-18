package links

import (
	"errors"
	"strings"
	"testing"

	core_errors "github.com/horizoonn/shortener/internal/errors"
)

func TestValidateOriginalURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		rawURL    string
		wantURL   string
		wantError bool
	}{
		{
			name:    "http URL",
			rawURL:  "http://example.com/path?q=1#fragment",
			wantURL: "http://example.com/path?q=1#fragment",
		},
		{
			name:    "https URL",
			rawURL:  "https://example.com",
			wantURL: "https://example.com",
		},
		{
			name:      "empty URL",
			rawURL:    "",
			wantError: true,
		},
		{
			name:      "whitespace only",
			rawURL:    "   ",
			wantError: true,
		},
		{
			name:      "leading whitespace",
			rawURL:    " https://example.com",
			wantError: true,
		},
		{
			name:      "missing host",
			rawURL:    "https:///path",
			wantError: true,
		},
		{
			name:      "unsupported scheme",
			rawURL:    "ftp://example.com",
			wantError: true,
		},
		{
			name:      "missing scheme",
			rawURL:    "example.com/path",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotURL, err := ValidateOriginalURL(tt.rawURL)
			if tt.wantError {
				if !errors.Is(err, core_errors.ErrInvalidArgument) {
					t.Fatalf("expected invalid argument, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("validate original URL: %v", err)
			}
			if gotURL != tt.wantURL {
				t.Fatalf("expected URL %q, got %q", tt.wantURL, gotURL)
			}
		})
	}
}

func TestValidateCustomAlias(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		alias     string
		wantError bool
	}{
		{name: "letters", alias: "abc"},
		{name: "digits", alias: "123"},
		{name: "hyphen and underscore", alias: "my_alias-123"},
		{name: "min length", alias: "ab", wantError: true},
		{name: "max length", alias: strings.Repeat("a", CustomAliasMaxLength+1), wantError: true},
		{name: "reserved healthz", alias: "healthz", wantError: true},
		{name: "reserved uppercase api", alias: "API", wantError: true},
		{name: "slash", alias: "abc/def", wantError: true},
		{name: "space", alias: "abc def", wantError: true},
		{name: "query string", alias: "abc?x=1", wantError: true},
		{name: "fragment", alias: "abc#frag", wantError: true},
		{name: "unicode", alias: "абв", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateCustomAlias(tt.alias)
			if tt.wantError {
				if !errors.Is(err, core_errors.ErrInvalidArgument) {
					t.Fatalf("expected invalid argument, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("validate custom alias: %v", err)
			}
		})
	}
}
