package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"reconpilot/internal/config"
	"reconpilot/internal/domain"
)

func TestMongoDBOfflineFallback(t *testing.T) {
	cfg := &config.DatabaseConfig{
		URI:      "mongodb://127.0.0.1:27019", // Port not running
		Database: "test_reconpilot",
		Timeout:  500 * time.Millisecond,
	}

	ctx := context.Background()
	mongoRepo, err := NewMongoDB(ctx, cfg)
	if err != nil {
		t.Fatalf("expected NewMongoDB to tolerate offline host, got error: %v", err)
	}

	if mongoRepo == nil {
		t.Fatalf("expected non-nil MongoDB instance")
	}

	// 1. Scope operations offline tolerance
	scopes, err := mongoRepo.List(ctx)
	if err != nil {
		t.Errorf("expected no error from List() offline, got: %v", err)
	}
	if scopes == nil {
		t.Errorf("expected non-nil slice from List()")
	}

	scope, err := mongoRepo.GetByID(ctx, uuid.New())
	if err != nil {
		t.Errorf("expected no error from GetByID() offline, got: %v", err)
	}
	if scope != nil {
		t.Errorf("expected nil scope from offline GetByID()")
	}

	// 2. Scan job operations offline tolerance
	job, err := mongoRepo.GetJobByID(ctx, uuid.New())
	if err != nil {
		t.Errorf("expected no error from GetJobByID() offline, got: %v", err)
	}
	if job != nil {
		t.Errorf("expected nil job from offline GetJobByID()")
	}

	// 3. Asset and finding batch insert offline tolerance
	err = mongoRepo.BatchInsertAssets(ctx, []*domain.Asset{})
	if err != nil {
		t.Errorf("expected no error on empty BatchInsertAssets, got: %v", err)
	}

	err = mongoRepo.BatchInsertFindings(ctx, []*domain.Finding{})
	if err != nil {
		t.Errorf("expected no error on empty BatchInsertFindings, got: %v", err)
	}
}
