package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"reconpilot/internal/ai"
	"reconpilot/internal/config"
	"reconpilot/internal/matching"
	"reconpilot/internal/repository"
	"reconpilot/internal/service"
	"reconpilot/internal/transport/stream"
)

func setupTestRouter(t *testing.T) (*http.Handler, *config.Config) {
	ctx := context.Background()
	cfg := &config.Config{
		Auth: config.AuthConfig{
			APIKey: "test-secret-key",
		},
		Server: config.ServerConfig{
			Port: 8080,
		},
	}

	repo := repository.NewMockRepository(false)
	engine := matching.NewEngine(matching.DefaultEngineConfig())
	aiClient, err := ai.NewClient(ctx, "", "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	hub := stream.NewHub()
	reconSvc := service.NewReconciliationService(repo, engine, aiClient, hub)
	exportSvc := service.NewExportService()

	router := NewRouter(cfg, reconSvc, exportSvc)
	var handler http.Handler = router
	return &handler, cfg
}

func TestHealthCheck(t *testing.T) {
	handler, _ := setupTestRouter(t)

	for _, path := range []string{"/", "/healthz", "/health", "/api/v1/health"} {
		req, _ := http.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		(*handler).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Path %s: Expected 200 OK, got %d", path, w.Code)
		}
		if !strings.Contains(w.Body.String(), "UP") {
			t.Errorf("Path %s: Expected body to contain 'UP', got %s", path, w.Body.String())
		}
	}
}

func TestAuthMiddlewareProtection(t *testing.T) {
	handler, cfg := setupTestRouter(t)

	// Unauthorized without key
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/summary", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized without API key, got %d", w.Code)
	}

	// Authorized with header
	reqAuth, _ := http.NewRequest(http.MethodGet, "/api/v1/summary", nil)
	reqAuth.Header.Set("X-API-Key", cfg.Auth.APIKey)
	wAuth := httptest.NewRecorder()
	(*handler).ServeHTTP(wAuth, reqAuth)

	if wAuth.Code != http.StatusOK {
		t.Errorf("Expected 200 OK with valid API key, got %d (body: %s)", wAuth.Code, wAuth.Body.String())
	}

	// Authorized with query parameter (?api_key=)
	reqQuery, _ := http.NewRequest(http.MethodGet, "/api/v1/summary?api_key="+cfg.Auth.APIKey, nil)
	wQuery := httptest.NewRecorder()
	(*handler).ServeHTTP(wQuery, reqQuery)

	if wQuery.Code != http.StatusOK {
		t.Errorf("Expected 200 OK with query api_key, got %d", wQuery.Code)
	}
}

func TestGetTracesAndDiscrepanciesAPI(t *testing.T) {
	handler, cfg := setupTestRouter(t)

	// 1. Get Traces
	reqTraces, _ := http.NewRequest(http.MethodGet, "/api/v1/traces", nil)
	reqTraces.Header.Set("X-API-Key", cfg.Auth.APIKey)
	wTraces := httptest.NewRecorder()
	(*handler).ServeHTTP(wTraces, reqTraces)

	if wTraces.Code != http.StatusOK {
		t.Errorf("Expected 200 for /api/v1/traces, got %d", wTraces.Code)
	}

	// 2. Get Discrepancies
	reqDiscs, _ := http.NewRequest(http.MethodGet, "/api/v1/discrepancies", nil)
	reqDiscs.Header.Set("X-API-Key", cfg.Auth.APIKey)
	wDiscs := httptest.NewRecorder()
	(*handler).ServeHTTP(wDiscs, reqDiscs)

	if wDiscs.Code != http.StatusOK {
		t.Errorf("Expected 200 for /api/v1/discrepancies, got %d", wDiscs.Code)
	}

	// 3. Export CSV
	reqExport, _ := http.NewRequest(http.MethodGet, "/api/v1/export?format=csv", nil)
	reqExport.Header.Set("X-API-Key", cfg.Auth.APIKey)
	wExport := httptest.NewRecorder()
	(*handler).ServeHTTP(wExport, reqExport)

	if wExport.Code != http.StatusOK {
		t.Errorf("Expected 200 for /api/v1/export, got %d", wExport.Code)
	}
	if !strings.Contains(wExport.Header().Get("Content-Type"), "text/csv") {
		t.Errorf("Expected text/csv content type, got %s", wExport.Header().Get("Content-Type"))
	}
}
