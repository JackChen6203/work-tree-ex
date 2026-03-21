package sync

import (
	"github.com/gin-gonic/gin"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/sync/bootstrap", func(c *gin.Context) {
		response.NotImplemented(c, "GET /sync/bootstrap")
	})
}
