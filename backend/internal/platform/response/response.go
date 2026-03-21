package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
)

type APIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

type ErrorEnvelope struct {
	Error APIError `json:"error"`
}

func JSON(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"data": data})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func Error(c *gin.Context, status int, code, message string, details any) {
	requestID, _ := c.Get("requestID")
	c.JSON(status, ErrorEnvelope{
		Error: APIError{
			Code:      code,
			Message:   message,
			Details:   details,
			RequestID: toString(requestID),
		},
	})
}

func NotImplemented(c *gin.Context, feature string) {
	Error(c, http.StatusNotImplemented, perrors.CodeNotImplemented, "feature is not implemented yet", gin.H{"feature": feature})
}

func toString(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}
