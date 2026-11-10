package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"reconpilot/internal/ai"
	"reconpilot/internal/domain"
	"reconpilot/internal/orchestrator"
	"reconpilot/internal/repository"
	"reconpilot/internal/security"
	"reconpilot/internal/service"
	"reconpilot/pkg/validator"
)

// APIHandler coordinates REST request handlers for scopes, scans, findings, and reports.
type APIHandler struct {
	db           *repository.PostgresDB
	orchestrator *orchestrator.PipelineOrchestrator
	verifier     *security.DNSVerifier
	analyst      *ai.GeminiAnalyst
	exporter     *service.ExportService
}

// NewAPIHandler creates a new APIHandler.
func NewAPIHandler(
	db *repository.PostgresDB,
	orch *orchestrator.PipelineOrchestrator,
	analyst *ai.GeminiAnalyst,
) *APIHandler {
	return &APIHandler{
		db:           db,
		orchestrator: orch,
		verifier:     security.NewDNSVerifier(10 * time.Second),
		analyst:      analyst,
		exporter:     service.NewExportService(),
	}
}

// ==============================================================================
// 1. HEALTH CHECKS
// ==============================================================================

func (h *APIHandler) HealthCheck(c *gin.Context) {
	dbStatus := "connected"
	if h.db == nil || h.db.Pool == nil {
		dbStatus = "offline (mock mode active)"
	} else if err := h.db.Pool.Ping(c.Request.Context()); err != nil {
		dbStatus = fmt.Sprintf("degraded (%v)", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"service":   "reconpilot-backend",
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"database":  dbStatus,
		"version":   "1.0.0-industrial",
	})
}

// ==============================================================================
// 2. SCOPE HANDLERS
// ==============================================================================

type CreateScopeRequest struct {
	Name               string   `json:"name" binding:"required"`
	RootDomain         string   `json:"root_domain" binding:"required"`
	AllowedCIDRs       []string `json:"allowed_cidrs"`
	ExcludedCIDRs      []string `json:"excluded_cidrs"`
	ExcludedSubdomains []string `json:"excluded_subdomains"`
	RateLimitRPS       int      `json:"rate_limit_rps"`
}

func (h *APIHandler) CreateScope(c *gin.Context) {
	var req CreateScopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validator.ValidateFQDN(req.RootDomain); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid root domain: %v", err)})
		return
	}

	token, err := h.verifier.GenerateToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate verification token"})
		return
	}

	rps := req.RateLimitRPS
	if rps <= 0 {
		rps = 20
	}

	scope := &domain.Scope{
		ID:                 uuid.New(),
		Name:               req.Name,
		RootDomain:         req.RootDomain,
		VerificationStatus: domain.VerificationPending,
		VerificationToken:  token,
		AllowedCIDRs:       req.AllowedCIDRs,
		ExcludedCIDRs:      req.ExcludedCIDRs,
		ExcludedSubdomains: req.ExcludedSubdomains,
		RateLimitRPS:       rps,
	}

	if err := h.db.Create(c.Request.Context(), scope); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save scope: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, scope)
}

func (h *APIHandler) ListScopes(c *gin.Context) {
	scopes, err := h.db.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if scopes == nil {
		scopes = []*domain.Scope{}
	}
	c.JSON(http.StatusOK, scopes)
}

func (h *APIHandler) GetScope(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scope UUID"})
		return
	}

	scope, err := h.db.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if scope == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scope not found"})
		return
	}

	c.JSON(http.StatusOK, scope)
}

func (h *APIHandler) VerifyScope(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scope UUID"})
		return
	}

	scope, err := h.db.GetByID(c.Request.Context(), id)
	if err != nil || scope == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scope not found"})
		return
	}

	// In development or dry-run, auto-verify if forced query param ?mock=true
	if c.Query("mock") == "true" {
		_ = h.db.UpdateVerificationStatus(c.Request.Context(), id, domain.VerificationVerified)
		scope.VerificationStatus = domain.VerificationVerified
		c.JSON(http.StatusOK, gin.H{
			"verified": true,
			"status":   domain.VerificationVerified,
			"details":  "Target verified via development bypass token.",
		})
		return
	}

	verified, details, verifyErr := h.verifier.VerifyOwnership(
		c.Request.Context(), scope.RootDomain, scope.VerificationToken,
	)
	if verifyErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": verifyErr.Error()})
		return
	}

	if verified {
		_ = h.db.UpdateVerificationStatus(c.Request.Context(), id, domain.VerificationVerified)
		c.JSON(http.StatusOK, gin.H{
			"verified": true,
			"status":   domain.VerificationVerified,
			"details":  details,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"verified": false,
		"status":   domain.VerificationPending,
		"details":  details,
		"instruction": fmt.Sprintf(
			"Publish DNS TXT record with value %q at root domain or _reconpilot-challenge.%s",
			scope.VerificationToken, scope.RootDomain,
		),
	})
}

func (h *APIHandler) DeleteScope(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scope UUID"})
		return
	}

	if err := h.db.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// ==============================================================================
// 3. SCAN JOB HANDLERS
// ==============================================================================

type CreateScanRequest struct {
	ScopeID uuid.UUID          `json:"scope_id" binding:"required"`
	Profile domain.ScanProfile `json:"profile"`
	DryRun  *bool              `json:"dry_run"`
}

func (h *APIHandler) CreateScan(c *gin.Context) {
	var req CreateScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	scope, err := h.db.GetByID(c.Request.Context(), req.ScopeID)
	if err != nil || scope == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scope not found"})
		return
	}

	profile := req.Profile
	if profile == "" {
		profile = domain.ProfileStandard
	}

	dryRun := false
	if req.DryRun != nil {
		dryRun = *req.DryRun
	}

	job := &domain.ScanJob{
		ID:               uuid.New(),
		ScopeID:          scope.ID,
		Status:           domain.ScanQueued,
		Profile:          profile,
		CurrentStage:     domain.StageIdle,
		StageCheckpoints: make(map[domain.StageName]domain.StageCheckpoint),
		Metrics:          domain.ScanMetrics{},
		DryRun:           dryRun,
	}

	if err := h.db.CreateJob(c.Request.Context(), job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create scan job: %v", err)})
		return
	}

	// Launch async execution in orchestrator worker pool
	go func(jobID uuid.UUID) {
		_ = h.orchestrator.ExecuteScan(context.Background(), jobID)
	}(job.ID)

	c.JSON(http.StatusAccepted, job)
}

func (h *APIHandler) ListScans(c *gin.Context) {
	scopeIDStr := c.Query("scope_id")
	if scopeIDStr != "" {
		scopeID, err := uuid.Parse(scopeIDStr)
		if err == nil {
			jobs, err := h.db.ListJobsByScope(c.Request.Context(), scopeID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if jobs == nil {
				jobs = []*domain.ScanJob{}
			}
			c.JSON(http.StatusOK, jobs)
			return
		}
	}

	// Fetch all scopes and their scans
	scopes, err := h.db.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var allJobs []*domain.ScanJob
	for _, s := range scopes {
		jobs, _ := h.db.ListJobsByScope(c.Request.Context(), s.ID)
		allJobs = append(allJobs, jobs...)
	}

	if allJobs == nil {
		allJobs = []*domain.ScanJob{}
	}

	c.JSON(http.StatusOK, allJobs)
}

func (h *APIHandler) GetScan(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scan UUID"})
		return
	}

	job, err := h.db.GetJobByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scan job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

func (h *APIHandler) CancelScan(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scan UUID"})
		return
	}

	success := h.orchestrator.CancelScan(id)
	msg := "Cancellation signal sent."
	_ = h.db.UpdateStatus(c.Request.Context(), id, domain.ScanCancelled, &msg)

	c.JSON(http.StatusOK, gin.H{"cancelled": success})
}

// ==============================================================================
// 4. ARTIFACT & TELEMETRY QUERY HANDLERS
// ==============================================================================

func (h *APIHandler) GetScanAssets(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scan UUID"})
		return
	}

	assets, err := h.db.ListAssetsByScanJob(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if assets == nil {
		assets = []*domain.Asset{}
	}

	c.JSON(http.StatusOK, assets)
}

func (h *APIHandler) GetScanDrifts(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scan UUID"})
		return
	}

	drifts, err := h.db.ListDriftsByScanJob(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if drifts == nil {
		drifts = []*domain.AssetDrift{}
	}

	c.JSON(http.StatusOK, drifts)
}

func (h *APIHandler) GetScanFindings(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scan UUID"})
		return
	}

	findings, err := h.db.ListFindingsByScanJob(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if findings == nil {
		findings = []*domain.Finding{}
	}

	c.JSON(http.StatusOK, findings)
}

func (h *APIHandler) GetScanReport(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scan UUID"})
		return
	}

	report, err := h.db.GetAIReportByScanJob(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// If report doesn't exist yet, synthesize on-the-fly!
	if report == nil {
		job, _ := h.db.GetJobByID(c.Request.Context(), id)
		if job == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "scan job not found"})
			return
		}
		scope, _ := h.db.GetByID(c.Request.Context(), job.ScopeID)
		root := "Unknown"
		if scope != nil {
			root = scope.RootDomain
		}

		assets, _ := h.db.ListAssetsByScanJob(c.Request.Context(), id)
		findings, _ := h.db.ListFindingsByScanJob(c.Request.Context(), id)

		report, err = h.analyst.GenerateReport(c.Request.Context(), id, root, assets, findings, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to synthesize AI report: %v", err)})
			return
		}
		_ = h.db.CreateAIReport(c.Request.Context(), report)
	}

	c.JSON(http.StatusOK, report)
}

func (h *APIHandler) GenerateReport(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scan UUID"})
		return
	}

	job, err := h.db.GetJobByID(c.Request.Context(), id)
	if err != nil || job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scan job not found"})
		return
	}

	scope, _ := h.db.GetByID(c.Request.Context(), job.ScopeID)
	root := "Unknown"
	if scope != nil {
		root = scope.RootDomain
	}

	assets, _ := h.db.ListAssetsByScanJob(c.Request.Context(), id)
	findings, _ := h.db.ListFindingsByScanJob(c.Request.Context(), id)

	report, err := h.analyst.GenerateReport(c.Request.Context(), id, root, assets, findings, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_ = h.db.CreateAIReport(c.Request.Context(), report)
	c.JSON(http.StatusOK, report)
}

// ==============================================================================
// 5. EXPORT HANDLERS
// ==============================================================================

func (h *APIHandler) ExportMarkdown(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scan UUID"})
		return
	}

	job, _ := h.db.GetJobByID(c.Request.Context(), id)
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scan job not found"})
		return
	}

	scope, _ := h.db.GetByID(c.Request.Context(), job.ScopeID)
	assets, _ := h.db.ListAssetsByScanJob(c.Request.Context(), id)
	drifts, _ := h.db.ListDriftsByScanJob(c.Request.Context(), id)
	findings, _ := h.db.ListFindingsByScanJob(c.Request.Context(), id)
	report, _ := h.db.GetAIReportByScanJob(c.Request.Context(), id)

	md, err := h.exporter.ExportMarkdown(scope, job, assets, drifts, findings, report)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fileName := fmt.Sprintf("reconpilot-report-%s.md", id.String()[:8])
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(md))
}

func (h *APIHandler) ExportJSON(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scan UUID"})
		return
	}

	job, _ := h.db.GetJobByID(c.Request.Context(), id)
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scan job not found"})
		return
	}

	scope, _ := h.db.GetByID(c.Request.Context(), job.ScopeID)
	assets, _ := h.db.ListAssetsByScanJob(c.Request.Context(), id)
	drifts, _ := h.db.ListDriftsByScanJob(c.Request.Context(), id)
	findings, _ := h.db.ListFindingsByScanJob(c.Request.Context(), id)
	report, _ := h.db.GetAIReportByScanJob(c.Request.Context(), id)

	jsonBytes, err := h.exporter.ExportJSON(scope, job, assets, drifts, findings, report)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fileName := fmt.Sprintf("reconpilot-scan-%s.json", id.String()[:8])
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	c.Data(http.StatusOK, "application/json; charset=utf-8", jsonBytes)
}

