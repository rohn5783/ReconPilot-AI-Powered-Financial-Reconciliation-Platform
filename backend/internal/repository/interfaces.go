package repository

import (
	"context"

	"github.com/google/uuid"

	"reconpilot/internal/domain"
)

// ScopeRepository defines data access for target scopes.
type ScopeRepository interface {
	Create(ctx context.Context, scope *domain.Scope) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Scope, error)
	GetByRootDomain(ctx context.Context, rootDomain string) (*domain.Scope, error)
	Update(ctx context.Context, scope *domain.Scope) error
	UpdateVerificationStatus(ctx context.Context, id uuid.UUID, status domain.VerificationStatus) error
	List(ctx context.Context) ([]*domain.Scope, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ScanJobRepository defines data access for scan pipelines.
type ScanJobRepository interface {
	Create(ctx context.Context, job *domain.ScanJob) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ScanJob, error)
	GetLatestCompletedByScope(ctx context.Context, scopeID uuid.UUID, excludeJobID uuid.UUID) (*domain.ScanJob, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ScanStatus, errMessage *string) error
	UpdateCurrentStage(ctx context.Context, id uuid.UUID, stage domain.StageName) error
	SaveCheckpoint(ctx context.Context, id uuid.UUID, checkpoint domain.StageCheckpoint) error
	UpdateMetrics(ctx context.Context, id uuid.UUID, metrics domain.ScanMetrics) error
	ListByScope(ctx context.Context, scopeID uuid.UUID) ([]*domain.ScanJob, error)
}

// AssetRepository defines batch and query operations for discovered assets.
type AssetRepository interface {
	BatchInsert(ctx context.Context, assets []*domain.Asset) error
	ListByScanJob(ctx context.Context, scanJobID uuid.UUID) ([]*domain.Asset, error)
	ListByScope(ctx context.Context, scopeID uuid.UUID) ([]*domain.Asset, error)
}

// AssetDriftRepository defines data access for drift records.
type AssetDriftRepository interface {
	BatchInsert(ctx context.Context, drifts []*domain.AssetDrift) error
	ListByScanJob(ctx context.Context, scanJobID uuid.UUID) ([]*domain.AssetDrift, error)
}

// FindingRepository defines data access for vulnerability findings.
type FindingRepository interface {
	BatchInsert(ctx context.Context, findings []*domain.Finding) error
	ListByScanJob(ctx context.Context, scanJobID uuid.UUID) ([]*domain.Finding, error)
	ListByAsset(ctx context.Context, assetID uuid.UUID) ([]*domain.Finding, error)
}

// AIReportRepository defines data access for AI synthesized executive briefings.
type AIReportRepository interface {
	Create(ctx context.Context, report *domain.AIReport) error
	GetByScanJob(ctx context.Context, scanJobID uuid.UUID) (*domain.AIReport, error)
}

// TelemetryBroadcaster defines real-time event dissemination.
type TelemetryBroadcaster interface {
	Broadcast(ctx context.Context, event domain.TelemetryEvent) error
	Subscribe(ctx context.Context, scanJobID uuid.UUID) (<-chan domain.TelemetryEvent, func())
}

