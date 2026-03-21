package ai

import (
	"github.com/gin-gonic/gin"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.POST("/trips/:tripId/ai/plans", func(c *gin.Context) {
		response.NotImplemented(c, "POST /trips/{tripId}/ai/plans")
	})
	v1.GET("/trips/:tripId/ai/plans", func(c *gin.Context) {
		response.NotImplemented(c, "GET /trips/{tripId}/ai/plans")
	})
	v1.GET("/trips/:tripId/ai/plans/:planId", func(c *gin.Context) {
		response.NotImplemented(c, "GET /trips/{tripId}/ai/plans/{planId}")
	})
	v1.POST("/trips/:tripId/ai/plans/:planId/adopt", func(c *gin.Context) {
		response.NotImplemented(c, "POST /trips/{tripId}/ai/plans/{planId}/adopt")
	})
}
