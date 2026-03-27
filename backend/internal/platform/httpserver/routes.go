package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/admin"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/ai"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/auth"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/budget"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/itinerary"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/maps"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/notifications"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/search"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/sync"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/trips"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/users"
)

func registerRoutes(engine *gin.Engine) {
	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	engine.GET("/readyz", func(c *gin.Context) {
		probe := getReadinessProbe()
		if probe == nil {
			c.JSON(http.StatusOK, gin.H{"status": "ready"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := probe(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"error":  "dependency unavailable",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ready"})
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
	search.RegisterRoutes(v1)
	admin.RegisterRoutes(v1)
}
