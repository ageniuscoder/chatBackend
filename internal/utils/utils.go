package utils

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

type CustomErrorResponse struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

// List of common SQLite time formats to try parsing.
// Adjusted to match your DB format (no timezone, no fractional seconds).
var timeFormats = []string{
	"2006-01-02 15:04:05",        // Your DB format (YYYY-MM-DD HH:MM:SS)
	"2006-01-02 15:04:05.999999", // If fractional seconds ever appear
	time.RFC3339,                 // ISO 8601 (optional fallback)
	time.RFC3339Nano,             // ISO 8601 with nanoseconds (optional fallback)
}

// ParseTime parses a time string using multiple common formats.
// It returns a time.Time object, or a zero value if parsing fails.
func ParseTime(t string) time.Time {
	if t == "" {
		return time.Time{}
	}

	for _, format := range timeFormats {
		parsedTime, err := time.Parse(format, t)
		if err == nil {
			return parsedTime
		}
	}

	// Log a warning if no format matches.
	fmt.Printf("Warning: Could not parse time string %q with any known format.\n", t)
	return time.Time{}
}

func ValidationErr(err validator.ValidationErrors) []CustomErrorResponse {
	var errors []CustomErrorResponse
	for _, fieldErr := range err {
		errors = append(errors, CustomErrorResponse{
			Field:   fieldErr.Field(),
			Tag:     fieldErr.ActualTag(),
			Message: GetErrorMessage(fieldErr),
		})
	}
	return errors
}

func GetErrorMessage(fe validator.FieldError) string {
	switch fe.ActualTag() {
	case "required":
		return "This field is required."
	default:
		return "Unknown validation error."
	}
}
