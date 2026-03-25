package dto

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

// Validate is a shared validator instance.
var Validate = validator.New()

// FieldError represents a single field-level validation error.
type FieldError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

// BindAndValidate binds JSON body and runs struct-tag validation.
// On failure it writes a 400 error with field-level details and returns false.
func BindAndValidate(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return false
	}

	if err := Validate.Struct(target); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			fieldErrors := make([]FieldError, 0, len(validationErrors))
			for _, fe := range validationErrors {
				fieldErrors = append(fieldErrors, FieldError{
					Field:   toJSONFieldName(fe.Field()),
					Tag:     fe.Tag(),
					Message: fmt.Sprintf("failed on '%s' validation", fe.Tag()),
				})
			}
			response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "validation failed", gin.H{"fields": fieldErrors})
			return false
		}
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "validation failed", gin.H{"error": err.Error()})
		return false
	}

	return true
}

// toJSONFieldName converts Go struct field name to lowercase camelCase.
func toJSONFieldName(field string) string {
	if field == "" {
		return field
	}
	return strings.ToLower(field[:1]) + field[1:]
}
