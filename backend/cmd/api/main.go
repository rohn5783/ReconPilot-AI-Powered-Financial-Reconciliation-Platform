package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"reconpilot/internal/ai"
	"reconpilot/internal/config"
	"reconpilot/internal/orchestrator"
	"reconpilot/internal/repository"
	transportHttp "reconpilot/internal/transport/http"
	"reconpilot/internal/transport/stream"
	"reconpilot/pkg/runner"
)

func main() {
	log.Println("\x1b[35;1m[RECONPILOT]\x1b[0m Initializing EASM Platform & Autonomous Security Engine...")

	// 1. Load typed configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("\x1b[31m[CONFIG_ERROR]\x1b[0m Failed to load configuration: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Connect to Database (with offline / mock tolerance)
	db, err := repository.NewPostgresDB(ctx, &cfg.Database)
	if err != nil {
		log.Printf("\x1b[33m[WARN]\x1b[0m Database connection degraded: %v. Running in mock/offline mode.", err)
	} else {
		defer db.Close()
		log.Println("\x1b[32m[DB]\x1b[0m PostgreSQL connection pool initialized.")
	}

	// 3. Connect to Redis Pub/Sub Telemetry Hub
	hub := repository.NewRedisTelemetryHub(ctx, &cfg.Redis)
	log.Println("\x1b[32m[STREAM]\x1b[0m Telemetry streaming hub active.")

	// 4. Initialize Hardened Subprocess Runner & Modules
	safeRunner := runner.NewSafeRunner(cfg.Pipeline.StageTimeout, cfg.Pipeline.RingBufferSizeKB)

	// 5. Initialize Pipeline Orchestrator & AI Analyst
	orch := orchestrator.NewPipelineOrchestrator(cfg, db, hub, safeRunner)
	analyst := ai.NewGeminiAnalyst(&cfg.AI)
	if cfg.AI.GeminiAPIKey != "" {
		log.Printf("\x1b[32m[AI]\x1b[0m Google GenAI SDK initialized (model: %s).", cfg.AI.GeminiModel)
	} else {
		log.Println("\x1b[33m[AI:OFFLINE]\x1b[0m GEMINI_API_KEY not set; using deterministic rule-based synthesis.")
	}

	// 6. Build HTTP Handlers & Router
	apiHandler := transportHttp.NewAPIHandler(db, orch, analyst)
	streamHandler := stream.NewStreamHandler(hub)
	router := transportHttp.SetupRouter(cfg, apiHandler, streamHandler)

	// 7. Start HTTP Server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		log.Printf("\x1b[32;1m[SERVER_READY]\x1b[0m ReconPilot API listening at http://%s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("\x1b[31m[SERVER_FATAL]\x1b[0m ListenAndServe failed: %v", err)
		}
	}()

	// 8. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("\x1b[33m[SHUTDOWN]\x1b[0m Gracefully shutting down ReconPilot server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("\x1b[31m[SHUTDOWN_ERROR]\x1b[0m Server forced to terminate: %v", err)
	}

	log.Println("\x1b[32m[SHUTDOWN_COMPLETE]\x1b[0m ReconPilot exited cleanly.")
}

