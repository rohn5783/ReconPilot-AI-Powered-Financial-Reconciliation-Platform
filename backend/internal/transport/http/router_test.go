package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"reconpilot/internal/ai"
	"reconpilot/internal/config"
	"reconpilot/internal/domain"
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
	authSvc := service.NewAuthService(repo.Users(), "test-jwt-secret")

	router := NewRouter(cfg, reconSvc, exportSvc, authSvc)
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

func TestAuthRegistrationAndLogin(t *testing.T) {
	handler, _ := setupTestRouter(t)

	// 1. Register a new user
	regPayload := `{"email":"analyst@reconpilot.io","password":"secretPassword123","full_name":"Jane Doe","company_name":"Acme Fintech","role":"SENIOR_ANALYST"}`
	reqReg, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(regPayload))
	reqReg.Header.Set("Content-Type", "application/json")
	wReg := httptest.NewRecorder()
	(*handler).ServeHTTP(wReg, reqReg)

	if wReg.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created on register, got %d: %s", wReg.Code, wReg.Body.String())
	}

	var regResp domain.AuthResponse
	if err := json.Unmarshal(wReg.Body.Bytes(), &regResp); err != nil {
		t.Fatalf("Failed to parse register response: %v", err)
	}
	if regResp.Token == "" {
		t.Errorf("Expected non-empty token on registration")
	}
	if regResp.User.Email != "analyst@reconpilot.io" {
		t.Errorf("Expected user email 'analyst@reconpilot.io', got %s", regResp.User.Email)
	}

	// 2. Login with credentials
	loginPayload := `{"email":"analyst@reconpilot.io","password":"secretPassword123"}`
	reqLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginPayload))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	(*handler).ServeHTTP(wLogin, reqLogin)

	if wLogin.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on login, got %d: %s", wLogin.Code, wLogin.Body.String())
	}

	var loginResp domain.AuthResponse
	if err := json.Unmarshal(wLogin.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("Failed to parse login response: %v", err)
	}
	if loginResp.Token == "" {
		t.Errorf("Expected non-empty token on login")
	}

	// 3. Access protected /api/v1/auth/me using Bearer token
	reqMe, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+loginResp.Token)
	wMe := httptest.NewRecorder()
	(*handler).ServeHTTP(wMe, reqMe)

	if wMe.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for /api/v1/auth/me with Bearer token, got %d: %s", wMe.Code, wMe.Body.String())
	}

	var userMe domain.UserPublic
	if err := json.Unmarshal(wMe.Body.Bytes(), &userMe); err != nil {
		t.Fatalf("Failed to parse /auth/me response: %v", err)
	}
	if userMe.Email != "analyst@reconpilot.io" {
		t.Errorf("Expected email 'analyst@reconpilot.io', got %s", userMe.Email)
	}
	if userMe.FullName != "Jane Doe" {
		t.Errorf("Expected full name 'Jane Doe', got %s", userMe.FullName)
	}

	// 4. Invalid login attempt
	badLoginPayload := `{"email":"analyst@reconpilot.io","password":"wrongPassword"}`
	reqBad, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(badLoginPayload))
	reqBad.Header.Set("Content-Type", "application/json")
	wBad := httptest.NewRecorder()
	(*handler).ServeHTTP(wBad, reqBad)

	if wBad.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized for bad password, got %d", wBad.Code)
	}
}

func TestUploadHistoryAndDownloadAPI(t *testing.T) {
	handler, cfg := setupTestRouter(t)

	// 1. Seed demo dataset (creates upload records)
	reqSeed, _ := http.NewRequest(http.MethodPost, "/api/v1/seed", nil)
	reqSeed.Header.Set("X-API-Key", cfg.Auth.APIKey)
	wSeed := httptest.NewRecorder()
	(*handler).ServeHTTP(wSeed, reqSeed)

	if wSeed.Code != http.StatusOK {
		t.Fatalf("Expected 200 for seed data, got %d: %s", wSeed.Code, wSeed.Body.String())
	}

	// 2. Fetch upload history
	reqUploads, _ := http.NewRequest(http.MethodGet, "/api/v1/uploads", nil)
	reqUploads.Header.Set("X-API-Key", cfg.Auth.APIKey)
	wUploads := httptest.NewRecorder()
	(*handler).ServeHTTP(wUploads, reqUploads)

	if wUploads.Code != http.StatusOK {
		t.Fatalf("Expected 200 for /api/v1/uploads, got %d", wUploads.Code)
	}

	var res struct {
		Data  []domain.UploadRecord `json:"data"`
		Total int                   `json:"total"`
	}
	if err := json.Unmarshal(wUploads.Body.Bytes(), &res); err != nil {
		t.Fatalf("Failed to parse uploads response: %v", err)
	}
	if len(res.Data) < 3 {
		t.Fatalf("Expected at least 3 uploaded files in history, got %d", len(res.Data))
	}

	// 3. Download the first uploaded file
	firstRecord := res.Data[0]
	reqDownload, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/uploads/%s/download", firstRecord.ID), nil)
	reqDownload.Header.Set("X-API-Key", cfg.Auth.APIKey)
	wDownload := httptest.NewRecorder()
	(*handler).ServeHTTP(wDownload, reqDownload)

	if wDownload.Code != http.StatusOK {
		t.Fatalf("Expected 200 for download, got %d", wDownload.Code)
	}
	if !strings.Contains(wDownload.Header().Get("Content-Type"), "text/csv") {
		t.Errorf("Expected text/csv content type, got %s", wDownload.Header().Get("Content-Type"))
	}
	if len(wDownload.Body.Bytes()) == 0 {
		t.Errorf("Expected non-empty downloaded file content")
	}
}
