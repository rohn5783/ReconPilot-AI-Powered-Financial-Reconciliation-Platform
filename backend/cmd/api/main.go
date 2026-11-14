package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"reconpilot/internal/ai"
	"reconpilot/internal/config"
	"reconpilot/internal/matching"
	"reconpilot/internal/repository"
	"reconpilot/internal/seed"
	"reconpilot/internal/service"
	transporthttp "reconpilot/internal/transport/http"
	"reconpilot/internal/transport/stream"
	"reconpilot/pkg/money"
)

func main() {
	log.Println("==============================================================================")
	log.Println(" ReconPilot: 3-Way Financial Reconciliation & AI Discrepancy Platform")
	log.Println(" Version: 1.0.0 | High-Throughput Golang + Exact Decimal Arithmetic Engine")
	log.Println("==============================================================================")

	// 1. Load typed environment configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[FATAL] Configuration load failure: %v", err)
	}
	log.Printf("[CONFIG] Port=%d | MockMode=%t | GeminiModel=%s | StrictSchema=%t",
		cfg.Server.Port, cfg.MockMode, cfg.AI.GeminiModel, cfg.Security.EnforceStrictSchema)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Initialize persistence repository (PostgreSQL or In-Memory Mock)
	var repo repository.ReconciliationRepository

	if cfg.MockMode || cfg.Database.URL == "" {
		log.Println("[STORAGE] MOCK_MODE=true or empty DATABASE_URL: Starting clean in-memory repository")
		repo = repository.NewMockRepository(false)
	} else {
		log.Printf("[STORAGE] Connecting to PostgreSQL database...")
		pgRepo, err := repository.NewPostgresRepository(ctx, cfg.Database.URL)
		if err != nil {
			log.Printf("[WARN] Failed to connect to PostgreSQL (%v). Seamlessly falling back to clean in-memory repository.", err)
			repo = repository.NewMockRepository(false)
		} else {
			log.Println("[STORAGE] Connected to PostgreSQL with connection pool successfully.")
			repo = pgRepo
		}
	}
	defer repo.Close()

	// 3. Initialize AI Diagnostician (Google GenAI Gemini 2.5 Flash + Fallback)
	aiClient, err := ai.NewClient(ctx, cfg.AI.GeminiAPIKey, cfg.AI.GeminiModel)
	if err != nil {
		log.Printf("[WARN] Gemini AI client initialization warning: %v. Using institutional fallback.", err)
	} else {
		log.Printf("[AI] Gemini AI Diagnostician initialized successfully.")
	}

	// 4. Initialize 3-Way Matching Engine with dynamic contract rates
	engineCfg := matching.EngineConfig{
		DefaultFeeRate: money.FromStringOrZero(cfg.Matching.ContractFeeRate),
		DefaultTaxRate: money.FromStringOrZero(cfg.Matching.ContractTaxRate),
		FeeTolerance:   money.FromStringOrZero(cfg.Matching.FeeTolerance),
		MaxConcurrency: cfg.Pipeline.MaxConcurrentBatches,
	}
	matchingEngine := matching.NewEngine(engineCfg)

	// 5. Initialize SSE Telemetry Broadcaster Hub
	hub := stream.NewHub()

	// 6. Initialize Orchestrator, Export and Auth Services
	reconSvc := service.NewReconciliationService(repo, matchingEngine, aiClient, hub)
	exportSvc := service.NewExportService()
	authSvc := service.NewAuthService(repo.Users(), cfg.Auth.JWTSecret)

	// 6.1. Prime reconciliation from existing database data (zero hardcoded auto-injection)
	if cfg.AutoSeedOnEmpty {
		log.Println("[STORAGE] AUTO_SEED_ON_EMPTY is enabled. Bootstrapping initial records...")
		_ = seed.BootstrapIfEmpty(ctx, repo, true, func(reconcileCtx context.Context) error {
			_, err := reconSvc.RunReconciliation(reconcileCtx, nil, nil, nil)
			return err
		})
	} else {
		// Run initial reconciliation on existing persistent records
		_, _ = reconSvc.RunReconciliation(ctx, nil, nil, nil)
	}

	// 7. Initialize Gin HTTP Router
	router := transporthttp.NewRouter(cfg, reconSvc, exportSvc, authSvc)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 8. Start server in goroutine
	go func() {
		log.Printf("[SERVER] ReconPilot HTTP & SSE API server listening on http://localhost:%d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] HTTP server failed to start: %v", err)
		}
	}()

	// 9. Graceful shutdown on OS interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("[SERVER] Shutting down ReconPilot server gracefully...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("[FATAL] Server forced to shutdown: %v", err)
	}

	log.Println("[SERVER] Server exited cleanly.")
}

