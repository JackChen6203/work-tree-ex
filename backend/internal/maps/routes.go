package maps

import (
	"github.com/gin-gonic/gin"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/maps/search", func(c *gin.Context) {
		response.NotImplemented(c, "GET /maps/search")
	})
	v1.POST("/maps/routes", func(c *gin.Context) {
		response.NotImplemented(c, "POST /maps/routes")
	})
}
