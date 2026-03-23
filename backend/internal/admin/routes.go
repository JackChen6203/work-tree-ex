package admin

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

// Job represents an AI planning job in the admin view.
type Job struct {
	ID          string     `json:"id"`
	TripID      string     `json:"tripId"`
	Status      string     `json:"status"`
	Provider    string     `json:"provider"`
	FailureCode *string    `json:"failureCode,omitempty"`
	QueuedAt    time.Time  `json:"queuedAt"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	FinishedAt  *time.Time `json:"finishedAt,omitempty"`
	RetryCount  int        `json:"retryCount"`
}

// AuditLog represents an admin audit log entry.
type AuditLog struct {
	ID           string         `json:"id"`
	ActorUserID  *string        `json:"actorUserId,omitempty"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resourceType"`
	ResourceID   string         `json:"resourceId"`
	BeforeState  map[string]any `json:"beforeState,omitempty"`
	AfterState   map[string]any `json:"afterState,omitempty"`
	RequestID    string         `json:"requestId,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
}

// ProviderHealth represents provider health status.
type ProviderHealth struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"` // healthy | degraded | down
	Latency   int       `json:"latencyMs"`
	ErrorRate float64   `json:"errorRate"`
	CheckedAt time.Time `json:"checkedAt"`
}

var (
	adminMu   sync.RWMutex
	adminJobs = []Job{
		{ID: uuid.NewString(), TripID: "demo-trip", Status: "succeeded", Provider: "openai", QueuedAt: time.Now().Add(-2 * time.Hour)},
		{ID: uuid.NewString(), TripID: "demo-trip", Status: "failed", Provider: "openai", FailureCode: strPtr("AI_PROVIDER_TIMEOUT"), QueuedAt: time.Now().Add(-1 * time.Hour)},
	}
	adminAuditLogs = []AuditLog{
		{ID: uuid.NewString(), Action: "adopt_ai_draft", ResourceType: "ai_plan_drafts", ResourceID: "demo-draft", CreatedAt: time.Now().Add(-30 * time.Minute)},
	}
)

func strPtr(s string) *string { return &s }

// RegisterRoutes registers admin API routes.
func RegisterRoutes(v1 *gin.RouterGroup) {
	admin := v1.Group("/admin")
	admin.GET("/jobs", listJobs)
	admin.POST("/jobs/:jobId/retry", retryJob)
	admin.POST("/jobs/:jobId/cancel", cancelJob)
	admin.GET("/providers/health", getProvidersHealth)
	admin.GET("/audit-logs", listAuditLogs)
}

func listJobs(c *gin.Context) {
	statusFilter := strings.TrimSpace(c.Query("status"))
	adminMu.RLock()
	items := make([]Job, 0, len(adminJobs))
	for _, j := range adminJobs {
		if statusFilter != "" && j.Status != statusFilter {
			continue
		}
		items = append(items, j)
	}
	adminMu.RUnlock()
	response.JSON(c, http.StatusOK, items)
}

func retryJob(c *gin.Context) {
	jobID := c.Param("jobId")

	adminMu.Lock()
	defer adminMu.Unlock()

	for i := range adminJobs {
		if adminJobs[i].ID != jobID {
			continue
		}
		if adminJobs[i].Status != "failed" {
			response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "only failed jobs can be retried", nil)
			return
		}
		adminJobs[i].Status = "queued"
		adminJobs[i].RetryCount++
		adminJobs[i].FailureCode = nil
		response.JSON(c, http.StatusOK, adminJobs[i])
		return
	}

	response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "job not found", gin.H{"jobId": jobID})
}

func cancelJob(c *gin.Context) {
	jobID := c.Param("jobId")

	adminMu.Lock()
	defer adminMu.Unlock()

	for i := range adminJobs {
		if adminJobs[i].ID != jobID {
			continue
		}
		if adminJobs[i].Status != "queued" && adminJobs[i].Status != "running" {
			response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "only queued or running jobs can be cancelled", nil)
			return
		}
		adminJobs[i].Status = "cancelled"
		now := time.Now().UTC()
		adminJobs[i].FinishedAt = &now
		response.JSON(c, http.StatusOK, adminJobs[i])
		return
	}

	response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "job not found", gin.H{"jobId": jobID})
}

func getProvidersHealth(c *gin.Context) {
	now := time.Now().UTC()
	health := []ProviderHealth{
		{Name: "openai", Status: "healthy", Latency: 450, ErrorRate: 0.01, CheckedAt: now},
		{Name: "anthropic", Status: "healthy", Latency: 380, ErrorRate: 0.005, CheckedAt: now},
		{Name: "google_maps", Status: "healthy", Latency: 120, ErrorRate: 0.002, CheckedAt: now},
	}
	response.JSON(c, http.StatusOK, health)
}

func listAuditLogs(c *gin.Context) {
	resourceType := strings.TrimSpace(c.Query("resourceType"))
	resourceID := strings.TrimSpace(c.Query("resourceId"))

	adminMu.RLock()
	items := make([]AuditLog, 0, len(adminAuditLogs))
	for _, log := range adminAuditLogs {
		if resourceType != "" && log.ResourceType != resourceType {
			continue
		}
		if resourceID != "" && log.ResourceID != resourceID {
			continue
		}
		items = append(items, log)
	}
	adminMu.RUnlock()

	response.JSON(c, http.StatusOK, items)
}
