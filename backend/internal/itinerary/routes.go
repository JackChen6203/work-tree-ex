package itinerary

import (
	"github.com/gin-gonic/gin"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/trips/:tripId/days", func(c *gin.Context) {
		response.NotImplemented(c, "GET /trips/{tripId}/days")
	})
	v1.POST("/trips/:tripId/items", func(c *gin.Context) {
		response.NotImplemented(c, "POST /trips/{tripId}/items")
	})
	v1.PATCH("/trips/:tripId/items/:itemId", func(c *gin.Context) {
		response.NotImplemented(c, "PATCH /trips/{tripId}/items/{itemId}")
	})
	v1.DELETE("/trips/:tripId/items/:itemId", func(c *gin.Context) {
		response.NotImplemented(c, "DELETE /trips/{tripId}/items/{itemId}")
	})
	v1.POST("/trips/:tripId/items:reorder", func(c *gin.Context) {
		response.NotImplemented(c, "POST /trips/{tripId}/items:reorder")
	})
}
