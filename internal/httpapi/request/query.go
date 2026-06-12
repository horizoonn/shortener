package request

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	core_errors "github.com/horizoonn/shortener/internal/errors"
)

const dateQueryLayout = "2006-01-02"

func GetDateQueryParam(r *http.Request, key string) (*time.Time, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse(dateQueryLayout, value)
	if err != nil {
		return nil, fmt.Errorf("query param %q=%q must use YYYY-MM-DD format: %v: %w", key, value, err, core_errors.ErrInvalidArgument)
	}

	return &parsed, nil
}

func GetIntQueryParam(r *http.Request, key string) (*int, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return nil, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf("query param %q=%q must be an integer: %v: %w", key, value, err, core_errors.ErrInvalidArgument)
	}

	return &parsed, nil
}
