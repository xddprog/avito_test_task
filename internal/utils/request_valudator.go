package utils

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
    "github.com/xddprog/avito-test-task/internal/entity" 
)


var validate = validator.New()



func ValidateForm(form any) error {
	if err := validate.Struct(form); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return NewValidationError(validationErrors)
		}
		return entity.ErrBadRequest 
	}
	return nil
}

func NewValidationError(errs validator.ValidationErrors) *entity.AppError {
	var errorMessages []string

	for _, err := range errs {
		msg := fmt.Sprintf("field '%s' failed on tag '%s'", err.Field(), err.Tag())
		errorMessages = append(errorMessages, msg)
	}
    
	return &entity.AppError{
		Code:     http.StatusBadRequest,
		SafeCode: "BAD_REQUEST",
		Message:  strings.Join(errorMessages, "; "),
	}
}