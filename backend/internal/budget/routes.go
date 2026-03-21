package budget

import (
	"github.com/gin-gonic/gin"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/trips/:tripId/budget", func(c *gin.Context) {
		response.NotImplemented(c, "GET /trips/{tripId}/budget")
	})
	v1.PUT("/trips/:tripId/budget", func(c *gin.Context) {
		response.NotImplemented(c, "PUT /trips/{tripId}/budget")
	})
	v1.GET("/trips/:tripId/expenses", func(c *gin.Context) {
		response.NotImplemented(c, "GET /trips/{tripId}/expenses")
	})
	v1.POST("/trips/:tripId/expenses", func(c *gin.Context) {
		response.NotImplemented(c, "POST /trips/{tripId}/expenses")
	})
}
