package http

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"

	"reconpilot/internal/config"
	"reconpilot/internal/transport/stream"
)

// SetupRouter constructs the hardened Gin engine and configures all endpoints.
func SetupRouter(
	cfg *config.Config,
	apiHandler *APIHandler,
	streamHandler *stream.StreamHandler,
) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()

	// Essential defensive middlewares
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(SecurityHeadersMiddleware())
	r.Use(CORSMiddleware(cfg.Server.CORSAllowedOrigins))
	r.Use(MetricsMiddleware())

	ipLimiter := NewIPRateLimiter(rate.Limit(50), 100)
	r.Use(RateLimitMiddleware(ipLimiter))

	// Health and Prometheus observability
	r.GET("/healthz", apiHandler.HealthCheck)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Authenticated API v1 routes
	apiV1 := r.Group("/api/v1")
	apiV1.Use(AuthMiddleware(&cfg.Auth, cfg.Environment))
	{
		// 1. Scopes
		scopes := apiV1.Group("/scopes")
		{
			scopes.POST("", apiHandler.CreateScope)
			scopes.GET("", apiHandler.ListScopes)
			scopes.GET("/:id", apiHandler.GetScope)
			scopes.POST("/:id/verify", apiHandler.VerifyScope)
			scopes.DELETE("/:id", apiHandler.DeleteScope)
		}

		// 2. Scans
		scans := apiV1.Group("/scans")
		{
			scans.POST("", apiHandler.CreateScan)
			scans.GET("", apiHandler.ListScans)
			scans.GET("/:id", apiHandler.GetScan)
			scans.POST("/:id/cancel", apiHandler.CancelScan)

			// Artifact queries
			scans.GET("/:id/assets", apiHandler.GetScanAssets)
			scans.GET("/:id/drifts", apiHandler.GetScanDrifts)
			scans.GET("/:id/findings", apiHandler.GetScanFindings)

			// AI Reports
			scans.GET("/:id/report", apiHandler.GetScanReport)
			scans.POST("/:id/report/generate", apiHandler.GenerateReport)

			// Export formats
			scans.GET("/:id/export/markdown", apiHandler.ExportMarkdown)
			scans.GET("/:id/export/json", apiHandler.ExportJSON)

			// Real-time telemetry streaming
			scans.GET("/:id/stream", streamHandler.HandleWebSocket)
			scans.GET("/:id/stream/sse", streamHandler.HandleSSE)
		}
	}

	// Serve Static UI Assets with Single-Page Application (SPA) fallback
	staticCandidates := []string{
		"./frontend/dist",
		"../frontend/dist",
		"../../frontend/dist",
		"/var/www/reconpilot",
	}
	for _, distPath := range staticCandidates {
		if fi, err := os.Stat(distPath); err == nil && fi.IsDir() {
			r.Static("/assets", filepath.Join(distPath, "assets"))
			r.NoRoute(func(c *gin.Context) {
				if strings.HasPrefix(c.Request.URL.Path, "/api/") {
					c.JSON(404, gin.H{"error": "API route not found"})
					return
				}
				c.File(filepath.Join(distPath, "index.html"))
			})
			break
		}
	}

	return r
}

