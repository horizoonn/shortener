package request

import (
	"fmt"
	"net/http"

	core_errors "github.com/horizoonn/shortener/internal/errors"
)

func GetStringPathValue(r *http.Request, key string) (string, error) {
	value := r.PathValue(key)
	if value == "" {
		return "", fmt.Errorf("no key=%q in path values: %w", key, core_errors.ErrInvalidArgument)
	}

	return value, nil
}
