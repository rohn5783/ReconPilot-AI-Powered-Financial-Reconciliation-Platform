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
	router.GET("/api/v1/health", handler.HealthCheck)

	// Public Authentication Routes
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/register", handler.Register)
		auth.POST("/login", handler.Login)
	}

	// Protected API Routes
	v1 := router.Group("/api/v1")
	v1.Use(AuthMiddleware(cfg.Auth.APIKey, authSvc))
	{
		// Operator Profile
		v1.GET("/auth/me", handler.GetCurrentUser)

		// Executive Workstation Dashboard
		v1.GET("/summary", handler.GetSummary)
		v1.GET("/traces", handler.GetTraces)
		v1.GET("/payout-batches", handler.GetPayoutBatches)

		// Discrepancy Auditing & Gemini AI Analysis
		v1.GET("/discrepancies", handler.GetDiscrepancies)
		v1.GET("/discrepancies/:id", handler.GetDiscrepancyByID)
		v1.POST("/discrepancies/:id/diagnose", handler.DiagnoseDiscrepancy)
		v1.PATCH("/discrepancies/:id/status", handler.UpdateDiscrepancyStatus)

		// Pipeline Execution & Realtime Telemetry Stream
		v1.POST("/reconcile", handler.TriggerReconciliation)
		v1.GET("/reconcile/stream", handler.StreamTelemetry)

		// Reset, Seed & Exports
		v1.POST("/reset", handler.ResetData)
		v1.POST("/seed", handler.SeedData)
		v1.GET("/export", handler.ExportReport)

		// Upload History & Ingestion Archive
		v1.GET("/uploads", handler.GetUploadHistory)
		v1.GET("/uploads/:id", handler.GetUploadRecordByID)
		v1.GET("/uploads/:id/download", handler.DownloadUploadedFile)
	}

	return router
}
