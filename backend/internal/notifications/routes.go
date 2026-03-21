package notifications

import (
	"github.com/gin-gonic/gin"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/notifications", func(c *gin.Context) {
		response.NotImplemented(c, "GET /notifications")
	})
	v1.POST("/notifications/:notificationId/read", func(c *gin.Context) {
		response.NotImplemented(c, "POST /notifications/{notificationId}/read")
	})
}
