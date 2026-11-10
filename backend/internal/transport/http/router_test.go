package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"reconpilot/internal/ai"
	"reconpilot/internal/config"
	"reconpilot/internal/orchestrator"
	"reconpilot/internal/repository"
	"reconpilot/internal/transport/stream"
	"reconpilot/pkg/runner"
)

func TestRouterHealthAndCORS(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		Server: config.ServerConfig{
			CORSAllowedOrigins: []string{"http://localhost:5173"},
		},
		Auth: config.AuthConfig{
			JWTSecret: "test-jwt-secret-12345678",
		},
	}

	hub := repository.NewRedisTelemetryHub(context.Background(), &cfg.Redis)
	safeRunner := runner.NewSafeRunner(10, 64)
	orch := orchestrator.NewPipelineOrchestrator(cfg, nil, hub, safeRunner)
	analyst := ai.NewGeminiAnalyst(&cfg.AI)
	apiHandler := NewAPIHandler(nil, orch, analyst)
	streamHandler := stream.NewStreamHandler(hub)

	router := SetupRouter(cfg, apiHandler, streamHandler)

	// 1. Health check test
	req, _ := http.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /healthz, got %d", w.Code)
	}

	// 2. CORS preflight test
	corsReq, _ := http.NewRequest("OPTIONS", "/api/v1/scopes", nil)
	corsReq.Header.Set("Origin", "http://localhost:5173")
	corsW := httptest.NewRecorder()
	router.ServeHTTP(corsW, corsReq)

	if corsW.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content for OPTIONS, got %d", corsW.Code)
	}
	if corsW.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Errorf("expected CORS origin header http://localhost:5173, got %s", corsW.Header().Get("Access-Control-Allow-Origin"))
	}

	// 3. Validation failure test on bad FQDN
	badScope := map[string]interface{}{
		"name":        "Invalid Test",
		"root_domain": "invalid-hostname-no-tld",
	}
	body, _ := json.Marshal(badScope)
	valReq, _ := http.NewRequest("POST", "/api/v1/scopes", bytes.NewReader(body))
	valReq.Header.Set("Content-Type", "application/json")
	valW := httptest.NewRecorder()
	router.ServeHTTP(valW, valReq)

	if valW.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for invalid domain, got %d", valW.Code)
	}
}

