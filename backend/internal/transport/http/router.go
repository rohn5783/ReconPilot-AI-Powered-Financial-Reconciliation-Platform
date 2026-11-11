package http

import (
	"github.com/gin-gonic/gin"

	"reconpilot/internal/config"
	"reconpilot/internal/service"
)

// NewRouter constructs and configures the Gin HTTP engine with secure middleware and API routes.
func NewRouter(cfg *config.Config, reconSvc *service.ReconciliationService, exportSvc *service.ExportService) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Global Middleware
	router.Use(LoggerMiddleware())
	router.Use(RecoveryMiddleware())
	router.Use(CORSMiddleware())

	handler := NewHandler(reconSvc, exportSvc)

	// Public Health Probes
	router.GET("/health", handler.HealthCheck)
	router.GET("/api/v1/health", handler.HealthCheck)

	// Protected API Routes
	v1 := router.Group("/api/v1")
	v1.Use(AuthMiddleware(cfg.Auth.APIKey))
	{
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

		// Seed & Exports
		v1.POST("/seed", handler.SeedData)
		v1.GET("/export", handler.ExportReport)
	}

	return router
}
