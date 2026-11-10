package modules

import (
	"context"
	"golang.org/x/time/rate"

	"reconpilot/internal/domain"
	"reconpilot/pkg/runner"
)

// StageContext provides shared execution state, channels, and limits to pipeline modules.
type StageContext struct {
	Context     context.Context
	ScanJob     *domain.ScanJob
	Scope       *domain.Scope
	Runner      *runner.SafeRunner
	Telemetry   chan<- domain.TelemetryEvent
	RateLimiter *rate.Limiter
	DryRun      bool

	// Inputs & state accumulated from preceding pipeline stages:
	DiscoveredHosts []string
	LiveAssets      []*domain.Asset
	OpenPortMap     map[string][]int
	Findings        []*domain.Finding
}

// StageResult captures the artifacts, assets, and telemetry summary produced by a stage.
type StageResult struct {
	Stage           domain.StageName
	Success         bool
	OutputCount     int
	DiscoveredHosts []string
	Assets          []*domain.Asset
	Findings        []*domain.Finding
	Metrics         map[string]interface{}
	Error           error
}

// StageModule defines the contract for pluggable reconnaissance and audit modules.
type StageModule interface {
	Name() domain.StageName
	ValidateInput(stageCtx *StageContext) error
	Execute(stageCtx *StageContext) (*StageResult, error)
}

