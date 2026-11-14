package http

import (
	"github.com/gin-gonic/gin"

	"reconpilot/internal/config"
	"reconpilot/internal/service"
)

// NewRouter constructs and configures the Gin HTTP engine with secure middleware and API routes.
func NewRouter(cfg *config.Config, reconSvc *service.ReconciliationService, exportSvc *service.ExportService, authSvc *service.AuthService) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Global Middleware
	router.Use(LoggerMiddleware())
	router.Use(RecoveryMiddleware())
	router.Use(CORSMiddleware())

	handler := NewHandler(reconSvc, exportSvc, authSvc)

	// Public Health Probes & Root Status (Prevents 404 on Root Checks)
	router.GET("/", handler.HealthCheck)
	router.GET("/healthz", handler.HealthCheck)
	router.GET("/health", handler.HealthCheck)
	router.GET("/api/health", handler.HealthCheck)
	router.GET("/api/v1/health", handler.HealthCheck)

	// Public Authentication Routes (support /api/v1/auth, /api/auth, and /auth with optional trailing slash)
	for _, prefix := range []string{"/api/v1/auth", "/api/auth", "/auth"} {
		auth := router.Group(prefix)
		{
			auth.POST("/register", handler.Register)
			auth.POST("/register/", handler.Register)
			auth.POST("/login", handler.Login)
			auth.POST("/login/", handler.Login)
		}
	}

	// Register core API routes under both /api/v1 and /api for maximum reverse proxy compatibility
	for _, apiPrefix := range []string{"/api/v1", "/api"} {
		apiGroup := router.Group(apiPrefix)
		apiGroup.Use(AuthMiddleware(cfg.Auth.APIKey, authSvc))
		{
			// Operator Profile
			apiGroup.GET("/auth/me", handler.GetCurrentUser)
			apiGroup.GET("/auth/me/", handler.GetCurrentUser)

			// Executive Workstation Dashboard
			apiGroup.GET("/summary", handler.GetSummary)
			apiGroup.GET("/traces", handler.GetTraces)
			apiGroup.GET("/payout-batches", handler.GetPayoutBatches)

			// Discrepancy Auditing & Gemini AI Analysis
			apiGroup.GET("/discrepancies", handler.GetDiscrepancies)
			apiGroup.GET("/discrepancies/:id", handler.GetDiscrepancyByID)
			apiGroup.POST("/discrepancies/:id/diagnose", handler.DiagnoseDiscrepancy)
			apiGroup.PATCH("/discrepancies/:id/status", handler.UpdateDiscrepancyStatus)

			// Pipeline Execution & Realtime Telemetry Stream
			apiGroup.POST("/reconcile", handler.TriggerReconciliation)
			apiGroup.GET("/reconcile/stream", handler.StreamTelemetry)

			// Reset, Seed & Exports
			apiGroup.POST("/reset", handler.ResetData)
			apiGroup.POST("/seed", handler.SeedData)
			apiGroup.GET("/export", handler.ExportReport)

			// Upload History & Ingestion Archive
			apiGroup.GET("/uploads", handler.GetUploadHistory)
			apiGroup.GET("/uploads/:id", handler.GetUploadRecordByID)
			apiGroup.GET("/uploads/:id/download", handler.DownloadUploadedFile)

			// Contract Rules & Dynamic Matching Configuration (MDR, GST, Fee Tolerance, Rounding)
			apiGroup.GET("/contract-rules", handler.GetContractRules)
			apiGroup.POST("/contract-rules", handler.CreateContractRule)
			apiGroup.PUT("/contract-rules/:id", handler.UpdateContractRule)
			apiGroup.POST("/contract-rules/:id/activate", handler.ActivateContractRule)
			apiGroup.DELETE("/contract-rules/:id", handler.DeleteContractRule)
		}
	}

	return router
}
