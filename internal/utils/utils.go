package utils

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

// The constant below is a common format for SQLite timestamps, which often include fractional seconds.
// `time.RFC3339` is a good alternative if your data includes timezone info.
const DBTimeFormat = "2006-01-02 15:04:05.999" // .999 handles fractional seconds up to nanoseconds

type CustomErrorResponse struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
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

// ParseTime parses a time string.
func ParseTime(t string) time.Time {
	if t == "" {
		return time.Time{}
	}
	// Try to parse with the DBTimeFormat first
	parsedTime, err := time.Parse(DBTimeFormat, t)
	if err == nil {
		return parsedTime
	}

	// If that fails, try a different, common format, such as RFC3339.
	// This makes the function more robust.
	parsedTime, err = time.Parse(time.RFC3339, t)
	if err == nil {
		return parsedTime
	}

	// If all else fails, log the error and return a zero time.
	fmt.Printf("Error parsing time string %q with multiple formats: %v\n", t, err)
	return time.Time{}
}
