package users

import (
	"github.com/gin-gonic/gin"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

func RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/me", func(c *gin.Context) {
		response.NotImplemented(c, "GET /users/me")
	})
	group.PATCH("/me", func(c *gin.Context) {
		response.NotImplemented(c, "PATCH /users/me")
	})
	group.GET("/me/preferences", func(c *gin.Context) {
		response.NotImplemented(c, "GET /users/me/preferences")
	})
	group.PUT("/me/preferences", func(c *gin.Context) {
		response.NotImplemented(c, "PUT /users/me/preferences")
	})
	group.GET("/me/llm-providers", func(c *gin.Context) {
		response.NotImplemented(c, "GET /users/me/llm-providers")
	})
	group.POST("/me/llm-providers", func(c *gin.Context) {
		response.NotImplemented(c, "POST /users/me/llm-providers")
	})
}
