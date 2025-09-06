package utils

import "github.com/go-playground/validator/v10"

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
