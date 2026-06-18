package request

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"
	core_errors "github.com/horizoonn/shortener/internal/errors"
)

var requestValidator = validator.New(validator.WithRequiredStructEnabled())

type validatable interface {
	Validate() error
}

func DecodeLimitedJSON(w http.ResponseWriter, r *http.Request, dst any, limit int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	return DecodeJSON(r, dst)
}

func DecodeAndValidateJSON(w http.ResponseWriter, r *http.Request, dst any, limit int64) error {
	if err := DecodeLimitedJSON(w, r, dst, limit); err != nil {
		return err
	}

	var err error
	if v, ok := dst.(validatable); ok {
		err = v.Validate()
	} else {
		err = requestValidator.Struct(dst)
	}
	if err != nil {
		return fmt.Errorf("validate json body: %w: %w", err, core_errors.ErrInvalidArgument)
	}

	return nil
}

func DecodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("decode json body: %w: %w", err, core_errors.ErrInvalidArgument)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode json body: multiple JSON values: %w", core_errors.ErrInvalidArgument)
		}

		return fmt.Errorf("decode json body trailing data: %w: %w", err, core_errors.ErrInvalidArgument)
	}
	return nil
}
