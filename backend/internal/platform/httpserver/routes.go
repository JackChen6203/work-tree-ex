package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/ai"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/auth"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/budget"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/itinerary"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/maps"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/notifications"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/sync"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/trips"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/users"
)

func registerRoutes(engine *gin.Engine) {
	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := engine.Group("/api/v1")
	auth.RegisterRoutes(v1.Group("/auth"))
	users.RegisterRoutes(v1.Group("/users"))
	trips.RegisterRoutes(v1)
	itinerary.RegisterRoutes(v1)
	budget.RegisterRoutes(v1)
	maps.RegisterRoutes(v1)
	ai.RegisterRoutes(v1)
	notifications.RegisterRoutes(v1)
	sync.RegisterRoutes(v1)
}
