package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

func RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/session", getSession)
	group.POST("/logout", logout)
}

func getSession(c *gin.Context) {
	response.JSON(c, http.StatusOK, gin.H{"user": nil, "roles": []string{}})
}

func logout(c *gin.Context) {
	response.NoContent(c)
}
