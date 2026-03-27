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
	Disabled  bool      `json:"disabled"`
}

// FeatureFlag represents a feature toggle.
type FeatureFlag struct {
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// SuspiciousUsage represents a flagged user.
type SuspiciousUsage struct {
	UserID       string `json:"userId"`
	RequestCount int    `json:"requestCount"`
	Reason       string `json:"reason"`
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

	// Feature flags
	featureFlags = map[string]*FeatureFlag{
		"ai_planner":         {Name: "ai_planner", Enabled: true, UpdatedAt: time.Now().UTC()},
		"budget_sync":        {Name: "budget_sync", Enabled: true, UpdatedAt: time.Now().UTC()},
		"push_notifications": {Name: "push_notifications", Enabled: true, UpdatedAt: time.Now().UTC()},
	}

	// Provider disable state
	providerDisabled = map[string]bool{}
)

func strPtr(s string) *string { return &s }

// AdminAuthGuard checks for X-Admin-Token header.
func AdminAuthGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimSpace(c.GetHeader("X-Admin-Token"))
		if token == "" || token != "admin-secret-token" {
			response.Error(c, http.StatusForbidden, perrors.CodeAdminForbidden,
				"admin access requires valid X-Admin-Token", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}

// RegisterRoutes registers admin API routes.
func RegisterRoutes(v1 *gin.RouterGroup) {
	admin := v1.Group("/admin")
	admin.Use(AdminAuthGuard())

	admin.GET("/jobs", listJobs)
	admin.POST("/jobs/:jobId/retry", retryJob)
	admin.POST("/jobs/:jobId/cancel", cancelJob)
	admin.GET("/providers/health", getProvidersHealth)
	admin.GET("/audit-logs", listAuditLogs)
	admin.GET("/usage/suspicious", getSuspiciousUsage)
	admin.POST("/feature-flags/:flag/toggle", toggleFeatureFlag)
	admin.GET("/feature-flags", listFeatureFlags)
	admin.POST("/providers/:name/disable", disableProvider)
	admin.POST("/providers/:name/enable", enableProvider)
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

		// Audit log
		if err := appendAuditLog(c, AuditLog{
			ID: uuid.NewString(), Action: "retry_job", ResourceType: "jobs",
			ResourceID: jobID, CreatedAt: time.Now().UTC(),
		}); err != nil {
			if response.DatabaseUnavailable(c, err) {
				return
			}
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to write audit log", nil)
			return
		}

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

		if err := appendAuditLog(c, AuditLog{
			ID: uuid.NewString(), Action: "cancel_job", ResourceType: "jobs",
			ResourceID: jobID, CreatedAt: now,
		}); err != nil {
			if response.DatabaseUnavailable(c, err) {
				return
			}
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to write audit log", nil)
			return
		}

		response.JSON(c, http.StatusOK, adminJobs[i])
		return
	}

	response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "job not found", gin.H{"jobId": jobID})
}

func getProvidersHealth(c *gin.Context) {
	now := time.Now().UTC()
	adminMu.RLock()
	defer adminMu.RUnlock()

	health := []ProviderHealth{
		{Name: "openai", Status: providerStatus("openai"), Latency: 450, ErrorRate: 0.01, CheckedAt: now, Disabled: providerDisabled["openai"]},
		{Name: "anthropic", Status: providerStatus("anthropic"), Latency: 380, ErrorRate: 0.005, CheckedAt: now, Disabled: providerDisabled["anthropic"]},
		{Name: "google_maps", Status: providerStatus("google_maps"), Latency: 120, ErrorRate: 0.002, CheckedAt: now, Disabled: providerDisabled["google_maps"]},
	}
	response.JSON(c, http.StatusOK, health)
}

func providerStatus(name string) string {
	if providerDisabled[name] {
		return "down"
	}
	return "healthy"
}

func listAuditLogs(c *gin.Context) {
	resourceType := strings.TrimSpace(c.Query("resourceType"))
	resourceID := strings.TrimSpace(c.Query("resourceId"))

	if getPool() != nil {
		items, err := listAuditLogsPostgres(c.Request.Context(), resourceType, resourceID)
		if err != nil {
			if response.DatabaseUnavailable(c, err) {
				return
			}
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to list audit logs", nil)
			return
		}
		response.JSON(c, http.StatusOK, items)
		return
	}

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

func getSuspiciousUsage(c *gin.Context) {
	// Mock: return sample suspicious users
	suspicious := []SuspiciousUsage{
		{UserID: "user-flagged-1", RequestCount: 5000, Reason: "abnormally high API request rate (>1000/hr)"},
		{UserID: "user-flagged-2", RequestCount: 3200, Reason: "excessive AI planning requests (>50/day)"},
	}
	response.JSON(c, http.StatusOK, suspicious)
}

func toggleFeatureFlag(c *gin.Context) {
	flagName := strings.TrimSpace(c.Param("flag"))

	adminMu.Lock()
	defer adminMu.Unlock()

	flag, ok := featureFlags[flagName]
	if !ok {
		response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "feature flag not found", gin.H{"flag": flagName})
		return
	}

	flag.Enabled = !flag.Enabled
	flag.UpdatedAt = time.Now().UTC()

	// Audit log
	if err := appendAuditLog(c, AuditLog{
		ID: uuid.NewString(), Action: "toggle_feature_flag", ResourceType: "feature_flags",
		ResourceID:  flagName,
		BeforeState: map[string]any{"enabled": !flag.Enabled},
		AfterState:  map[string]any{"enabled": flag.Enabled},
		CreatedAt:   flag.UpdatedAt,
	}); err != nil {
		if response.DatabaseUnavailable(c, err) {
			return
		}
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to write audit log", nil)
		return
	}

	response.JSON(c, http.StatusOK, flag)
}

func listFeatureFlags(c *gin.Context) {
	adminMu.RLock()
	defer adminMu.RUnlock()

	flags := make([]*FeatureFlag, 0, len(featureFlags))
	for _, f := range featureFlags {
		flags = append(flags, f)
	}
	response.JSON(c, http.StatusOK, flags)
}

func disableProvider(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))

	adminMu.Lock()
	defer adminMu.Unlock()

	providerDisabled[name] = true

	if err := appendAuditLog(c, AuditLog{
		ID: uuid.NewString(), Action: "disable_provider", ResourceType: "providers",
		ResourceID: name, CreatedAt: time.Now().UTC(),
	}); err != nil {
		if response.DatabaseUnavailable(c, err) {
			return
		}
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to write audit log", nil)
		return
	}

	response.JSON(c, http.StatusOK, gin.H{"provider": name, "disabled": true})
}

func enableProvider(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))

	adminMu.Lock()
	defer adminMu.Unlock()

	delete(providerDisabled, name)

	if err := appendAuditLog(c, AuditLog{
		ID: uuid.NewString(), Action: "enable_provider", ResourceType: "providers",
		ResourceID: name, CreatedAt: time.Now().UTC(),
	}); err != nil {
		if response.DatabaseUnavailable(c, err) {
			return
		}
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to write audit log", nil)
		return
	}

	response.JSON(c, http.StatusOK, gin.H{"provider": name, "disabled": false})
}

func appendAuditLog(c *gin.Context, entry AuditLog) error {
	if getPool() != nil {
		_, err := createAuditLogPostgres(c.Request.Context(), entry)
		return err
	}
	adminAuditLogs = append(adminAuditLogs, entry)
	return nil
}
