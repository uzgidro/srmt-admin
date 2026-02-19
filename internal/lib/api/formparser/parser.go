package formparser

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// GetFormInt64 safely parses int64 from form field
// Returns nil if field is not present or empty
func GetFormInt64(r *http.Request, key string) (*int64, error) {
	value := r.FormValue(key)
	if value == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", key, err)
	}

	return &parsed, nil
}

// GetFormInt64Required parses required int64 from form field
func GetFormInt64Required(r *http.Request, key string) (int64, error) {
	value := r.FormValue(key)
	if value == "" {
		return 0, fmt.Errorf("%s is required", key)
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}

	return parsed, nil
}

// GetFormFloat64 safely parses float64 from form field
// Returns nil if field is not present or empty
func GetFormFloat64(r *http.Request, key string) (*float64, error) {
	value := r.FormValue(key)
	if value == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", key, err)
	}

	return &parsed, nil
}

// GetFormBool safely parses bool from form field
// Returns nil if field is not present or empty
// Accepts: "true", "false", "1", "0"
func GetFormBool(r *http.Request, key string) (*bool, error) {
	value := r.FormValue(key)
	if value == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", key, err)
	}

	return &parsed, nil
}

// GetFormString safely gets string from form field
// Returns nil if field is not present or empty
func GetFormString(r *http.Request, key string) *string {
	value := r.FormValue(key)
	if value == "" {
		return nil
	}

	return &value
}

// GetFormStringRequired gets required string from form field
func GetFormStringRequired(r *http.Request, key string) (string, error) {
	value := r.FormValue(key)
	if value == "" {
		return "", fmt.Errorf("%s is required", key)
	}

	return value, nil
}

// GetFormTime safely parses time from form field using given layout
// Returns nil if field is not present or empty
func GetFormTime(r *http.Request, key string, layout string) (*time.Time, error) {
	value := r.FormValue(key)
	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse(layout, value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s format: %w", key, err)
	}

	return &parsed, nil
}

// GetFormTimeRequired parses required time from form field
func GetFormTimeRequired(r *http.Request, key string, layout string) (time.Time, error) {
	value := r.FormValue(key)
	if value == "" {
		return time.Time{}, fmt.Errorf("%s is required", key)
	}

	parsed, err := time.Parse(layout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid %s format: %w", key, err)
	}

	return parsed, nil
}

// GetFormTimePtr parses **time.Time from form field (for edit operations)
// Returns nil if field is not present
// Returns pointer to nil if field is empty string (to set NULL)
func GetFormTimePtr(r *http.Request, key string, layout string) (**time.Time, error) {
	if !r.Form.Has(key) {
		return nil, nil
	}

	value := r.FormValue(key)
	if value == "" {
		// Field is present but empty - means set to NULL
		var nilTime *time.Time = nil
		return &nilTime, nil
	}

	parsed, err := time.Parse(layout, value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s format: %w", key, err)
	}

	ptrTime := &parsed
	return &ptrTime, nil
}

// GetFormFileIDs parses comma-separated file IDs from form field
// Example: "1,2,3" -> [1, 2, 3]
// Returns empty slice if field is not present or empty
func GetFormFileIDs(r *http.Request, key string) ([]int64, error) {
	value := r.FormValue(key)
	if value == "" {
		return []int64{}, nil
	}

	parts := strings.Split(value, ",")
	ids := make([]int64, 0, len(parts))

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid file ID at position %d: %w", i, err)
		}

		ids = append(ids, id)
	}

	return ids, nil
}

// HasFormField checks if form field is present (even if empty)
func HasFormField(r *http.Request, key string) bool {
	return r.Form.Has(key)
}

// IsMultipartForm checks if request is multipart/form-data
func IsMultipartForm(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	return strings.HasPrefix(contentType, "multipart/form-data")
}

// IsJSONRequest checks if request is application/json
func IsJSONRequest(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	return strings.HasPrefix(contentType, "application/json")
}

// GetFormInt64Slice parses comma-separated int64 list from form field
func GetFormInt64Slice(r *http.Request, key string) ([]int64, error) {
	return GetFormFileIDs(r, key) // Re-use existing logic as it does exactly this
}

// GetFormDate parses date in "2006-01-02" format
func GetFormDate(r *http.Request, key string) (*time.Time, error) {
	return GetFormTime(r, key, time.DateOnly)
}

// GetFormDateRequired parses required date in "2006-01-02" format
func GetFormDateRequired(r *http.Request, key string) (time.Time, error) {
	return GetFormTimeRequired(r, key, time.DateOnly)
}

// GetFormDateTime parses datetime in RFC3339 format
func GetFormDateTime(r *http.Request, key string) (*time.Time, error) {
	return GetFormTime(r, key, time.RFC3339)
}

// GetFormDateTimeRequired parses required datetime in RFC3339 format
func GetFormDateTimeRequired(r *http.Request, key string) (time.Time, error) {
	return GetFormTimeRequired(r, key, time.RFC3339)
}

// GetFormFile retrieves a file from multipart form, validates existence
// Returns nil, nil if file is optional and missing (http.ErrMissingFile)
func GetFormFile(r *http.Request, key string) (*multipart.FileHeader, error) {
	_, fileHeader, err := r.FormFile(key)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get file %s: %w", key, err)
	}
	return fileHeader, nil
}

// GetURLParamInt64 parses int64 from chi URL param
func GetURLParamInt64(r *http.Request, key string) (int64, error) {
	valStr := chi.URLParam(r, key)
	if valStr == "" {
		return 0, fmt.Errorf("url param %s is required", key)
	}
	val, err := strconv.ParseInt(valStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s param: %w", key, err)
	}
	return val, nil
}

func GetFormDateInLocation(r *http.Request, key string, loc *time.Location) (*time.Time, error) {
	val := r.FormValue(key)
	if val == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", val, loc)
	if err != nil {
		return nil, fmt.Errorf("invalid %s param: %w", key, err)
	}
	return &t, nil
}
