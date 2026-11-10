package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/time/rate"

	"reconpilot/internal/config"
	"reconpilot/internal/domain"
	"reconpilot/internal/modules"
	"reconpilot/internal/modules/auditor"
	"reconpilot/internal/modules/discovery"
	"reconpilot/internal/modules/portscan"
	"reconpilot/internal/modules/prober"
	"reconpilot/internal/repository"
	"reconpilot/internal/service"
	"reconpilot/pkg/runner"
)

// PipelineOrchestrator manages the DAG execution lifecycle, bounded worker pools,
// stage checkpoint persistence, and live telemetry streaming.
type PipelineOrchestrator struct {
	cfg         *config.Config
	db          *repository.PostgresDB
	hub         *repository.RedisTelemetryHub
	runner      *runner.SafeRunner
	driftEngine *service.DriftEngine
	stageOrder  []domain.StageName
	modules     map[domain.StageName]modules.StageModule
	workerSem   chan struct{}
	mu          sync.Mutex
	activeScans map[uuid.UUID]context.CancelFunc

	// In-memory store for standalone CLI execution without database
	inMemoryJobs     map[uuid.UUID]*domain.ScanJob
	inMemoryScopes   map[uuid.UUID]*domain.Scope
	inMemoryAssets   map[uuid.UUID][]*domain.Asset
	inMemoryFindings map[uuid.UUID][]*domain.Finding
	inMemoryDrifts   map[uuid.UUID][]*domain.AssetDrift
}

// NewPipelineOrchestrator creates a fully wired orchestrator.
func NewPipelineOrchestrator(
	cfg *config.Config,
	db *repository.PostgresDB,
	hub *repository.RedisTelemetryHub,
	safeRunner *runner.SafeRunner,
) *PipelineOrchestrator {
	maxWorkers := cfg.Pipeline.MaxConcurrentScans
	if maxWorkers <= 0 {
		maxWorkers = 4
	}

	disc := discovery.NewDiscoveryModule(cfg.Pipeline.SubfinderBin)
	prob := prober.NewProberModule(cfg.Pipeline.HttpxBin)
	port := portscan.NewPortScanModule(cfg.Pipeline.NmapBin)
	audit := auditor.NewAuditorModule(cfg.Pipeline.NucleiBin)

	moduleMap := map[domain.StageName]modules.StageModule{
		domain.StageDiscovery: disc,
		domain.StageProbing:   prob,
		domain.StagePortScan:  port,
		domain.StageAuditor:   audit,
	}

	return &PipelineOrchestrator{
		cfg:         cfg,
		db:          db,
		hub:         hub,
		runner:      safeRunner,
		driftEngine: service.NewDriftEngine(),
		stageOrder: []domain.StageName{
			domain.StageDiscovery,
			domain.StageProbing,
			domain.StagePortScan,
			domain.StageAuditor,
		},
		modules:          moduleMap,
		workerSem:        make(chan struct{}, maxWorkers),
		activeScans:      make(map[uuid.UUID]context.CancelFunc),
		inMemoryJobs:     make(map[uuid.UUID]*domain.ScanJob),
		inMemoryScopes:   make(map[uuid.UUID]*domain.Scope),
		inMemoryAssets:   make(map[uuid.UUID][]*domain.Asset),
		inMemoryFindings: make(map[uuid.UUID][]*domain.Finding),
		inMemoryDrifts:   make(map[uuid.UUID][]*domain.AssetDrift),
	}
}

// RegisterStandaloneJob registers in-memory job and scope for standalone execution without DB.
func (o *PipelineOrchestrator) RegisterStandaloneJob(job *domain.ScanJob, scope *domain.Scope) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.inMemoryJobs[job.ID] = job
	o.inMemoryScopes[scope.ID] = scope
}

// GetJobArtifacts retrieves accumulated scan artifacts from DB or memory fallback.
func (o *PipelineOrchestrator) GetJobArtifacts(scanJobID uuid.UUID) ([]*domain.Asset, []*domain.Finding, []*domain.AssetDrift) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.inMemoryAssets[scanJobID], o.inMemoryFindings[scanJobID], o.inMemoryDrifts[scanJobID]
}

// CancelScan cancels an ongoing scan job context.
func (o *PipelineOrchestrator) CancelScan(scanJobID uuid.UUID) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	cancel, exists := o.activeScans[scanJobID]
	if exists {
		cancel()
		delete(o.activeScans, scanJobID)
		return true
	}
	return false
}

// IsScanning checks if a scan job is actively processing in the worker pool.
func (o *PipelineOrchestrator) IsScanning(scanJobID uuid.UUID) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	_, exists := o.activeScans[scanJobID]
	return exists
}

// ExecuteScan executes the sequential DAG stages with checkpointing, rate limiting, and telemetry.
func (o *PipelineOrchestrator) ExecuteScan(ctx context.Context, scanJobID uuid.UUID) error {
	// Acquire slot from bounded worker pool
	select {
	case o.workerSem <- struct{}{}:
		defer func() { <-o.workerSem }()
	case <-ctx.Done():
		return ctx.Err()
	}

	scanCtx, cancel := context.WithTimeout(ctx, o.cfg.Pipeline.GlobalScanTimeout)
	defer cancel()

	o.mu.Lock()
	o.activeScans[scanJobID] = cancel
	o.mu.Unlock()

	defer func() {
		o.mu.Lock()
		delete(o.activeScans, scanJobID)
		o.mu.Unlock()
	}()

	// 1. Load ScanJob and Scope
	var job *domain.ScanJob
	var scope *domain.Scope

	if o.db != nil && o.db.Pool != nil {
		job, _ = o.db.GetJobByID(scanCtx, scanJobID)
		if job != nil {
			scope, _ = o.db.GetByID(scanCtx, job.ScopeID)
		}
	}

	// In-memory fallback if DB not available or offline
	if job == nil {
		o.mu.Lock()
		job = o.inMemoryJobs[scanJobID]
		if job != nil {
			scope = o.inMemoryScopes[job.ScopeID]
		}
		o.mu.Unlock()
	}

	if job == nil {
		return fmt.Errorf("scan job %s not found in database or memory", scanJobID)
	}
	if scope == nil {
		return fmt.Errorf("scope %s not found for scan job", job.ScopeID)
	}

	// 2. Setup Telemetry Stream
	telemetryChan := make(chan domain.TelemetryEvent, 512)
	go func() {
		for ev := range telemetryChan {
			_ = o.hub.Broadcast(context.Background(), ev)
		}
	}()
	defer close(telemetryChan)

	now := time.Now()
	job.Status = domain.ScanRunning
	job.StartedAt = &now
	if o.db != nil && o.db.Pool != nil {
		_ = o.db.UpdateStatus(scanCtx, job.ID, domain.ScanRunning, nil)
	}

	o.emitLog(telemetryChan, job.ID, domain.StageIdle, fmt.Sprintf(
		"\x1b[35;1m[RECONPILOT_ORCHESTRATOR]\x1b[0m Starting DAG execution for \x1b[1m%s\x1b[0m (profile: %s, dry_run: %v)",
		scope.RootDomain, job.Profile, job.DryRun,
	))

	// Rate limiter for target requests
	rps := scope.RateLimitRPS
	if rps <= 0 {
		rps = 20
	}
	limiter := rate.NewLimiter(rate.Limit(rps), rps)

	// Shared stage context
	stageCtx := &modules.StageContext{
		Context:         scanCtx,
		ScanJob:         job,
		Scope:           scope,
		Runner:          o.runner,
		Telemetry:       telemetryChan,
		RateLimiter:     limiter,
		DryRun:          job.DryRun,
		OpenPortMap:     make(map[string][]int),
		DiscoveredHosts: []string{},
	}

	startTime := time.Now()

	// 3. Sequential Stage Execution with Checkpoint Recovery
	for _, stageName := range o.stageOrder {
		module, exists := o.modules[stageName]
		if !exists {
			continue
		}

		// Check if stage already completed in prior run (Checkpoint Recovery)
		if cp, ok := job.StageCheckpoints[stageName]; ok && cp.Status == domain.StageStatusDone {
			o.emitLog(telemetryChan, job.ID, stageName, fmt.Sprintf(
				"\x1b[33m[CHECKPOINT_RECOVER]\x1b[0m Skipping %s (already completed at %s)",
				stageName, cp.CompletedAt.Format(time.RFC3339),
			))
			// Restore prior state from DB
			o.restoreStageState(scanCtx, stageName, stageCtx)
			continue
		}

		// Stage Start Checkpoint
		stageStart := time.Now()
		if o.db != nil && o.db.Pool != nil {
			_ = o.db.UpdateCurrentStage(scanCtx, job.ID, stageName)
			_ = o.db.SaveCheckpoint(scanCtx, job.ID, domain.StageCheckpoint{
				Stage:     stageName,
				Status:    domain.StageStatusRunning,
				StartedAt: stageStart,
			})
		}

		o.emitEvent(telemetryChan, domain.TelemetryEvent{
			Type:      "stage_update",
			ScanJobID: job.ID,
			Stage:     stageName,
			Timestamp: stageStart,
			Payload:   map[string]interface{}{"status": "running"},
		})

		// Execute Stage Module
		res, err := module.Execute(stageCtx)
		stageDuration := time.Since(stageStart)

		if err != nil {
			errMsg := err.Error()
			if o.db != nil && o.db.Pool != nil {
				_ = o.db.SaveCheckpoint(scanCtx, job.ID, domain.StageCheckpoint{
					Stage:      stageName,
					Status:     domain.StageStatusFailed,
					StartedAt:  stageStart,
					DurationMs: stageDuration.Milliseconds(),
					Error:      &errMsg,
				})
				_ = o.db.UpdateStatus(scanCtx, job.ID, domain.ScanFailed, &errMsg)
			}

			o.emitLog(telemetryChan, job.ID, stageName, fmt.Sprintf(
				"\x1b[31;1m[STAGE_FAILED]\x1b[0m %s failed: %v", stageName, err,
			))
			return fmt.Errorf("stage %s failed: %w", stageName, err)
		}

		// Accumulate results into stageCtx for next stages
		if len(res.DiscoveredHosts) > 0 {
			stageCtx.DiscoveredHosts = res.DiscoveredHosts
		}
		if len(res.Assets) > 0 {
			stageCtx.LiveAssets = res.Assets
			if o.db != nil && o.db.Pool != nil {
				_ = o.db.BatchInsertAssets(scanCtx, res.Assets)
			}
			o.mu.Lock()
			o.inMemoryAssets[job.ID] = res.Assets
			o.mu.Unlock()
		}
		if len(res.Findings) > 0 {
			stageCtx.Findings = res.Findings
			if o.db != nil && o.db.Pool != nil {
				_ = o.db.BatchInsertFindings(scanCtx, res.Findings)
			}
			o.mu.Lock()
			o.inMemoryFindings[job.ID] = res.Findings
			o.mu.Unlock()
		}

		// Stage Success Checkpoint
		nowCompleted := time.Now()
		if o.db != nil && o.db.Pool != nil {
			_ = o.db.SaveCheckpoint(scanCtx, job.ID, domain.StageCheckpoint{
				Stage:       stageName,
				Status:      domain.StageStatusDone,
				StartedAt:   stageStart,
				CompletedAt: &nowCompleted,
				DurationMs:  stageDuration.Milliseconds(),
				OutputCount: res.OutputCount,
			})
		}

		o.emitEvent(telemetryChan, domain.TelemetryEvent{
			Type:      "stage_update",
			ScanJobID: job.ID,
			Stage:     stageName,
			Timestamp: nowCompleted,
			Payload: map[string]interface{}{
				"status":       "done",
				"output_count": res.OutputCount,
				"duration_ms":  stageDuration.Milliseconds(),
			},
		})
	}

	// 4. Historical Drift Diffing Engine
	o.emitLog(telemetryChan, job.ID, domain.StageIdle, "\x1b[36m[DRIFT_ANALYSIS]\x1b[0m Computing asset mutations against baseline scan...")

	var prevAssets []*domain.Asset
	var prevFindings []*domain.Finding
	var prevJobID *uuid.UUID

	if o.db != nil && o.db.Pool != nil {
		prevJob, _ := o.db.GetLatestCompletedByScope(scanCtx, scope.ID, job.ID)
		if prevJob != nil {
			prevJobID = &prevJob.ID
			prevAssets, _ = o.db.ListAssetsByScanJob(scanCtx, prevJob.ID)
			prevFindings, _ = o.db.ListFindingsByScanJob(scanCtx, prevJob.ID)
			o.emitLog(telemetryChan, job.ID, domain.StageIdle, fmt.Sprintf(
				"\x1b[32m[DRIFT_BASELINE]\x1b[0m Comparing with baseline scan %s (completed %s)",
				prevJob.ID, prevJob.CompletedAt.Format(time.RFC3339),
			))
		} else {
			o.emitLog(telemetryChan, job.ID, domain.StageIdle, "\x1b[33m[DRIFT_BASELINE]\x1b[0m No previous scan found. Establishing initial security baseline.")
		}
	} else {
		o.emitLog(telemetryChan, job.ID, domain.StageIdle, "\x1b[33m[DRIFT_BASELINE]\x1b[0m Standalone memory mode: establishing initial security baseline.")
	}

	drifts, driftAnalysis := o.driftEngine.ComputeDrift(
		job.ID, prevJobID, stageCtx.LiveAssets, prevAssets, stageCtx.Findings, prevFindings,
	)

	if len(drifts) > 0 {
		if o.db != nil && o.db.Pool != nil {
			_ = o.db.BatchInsertDrifts(scanCtx, drifts)
		}
		o.mu.Lock()
		o.inMemoryDrifts[job.ID] = drifts
		o.mu.Unlock()

		o.emitLog(telemetryChan, job.ID, domain.StageIdle, fmt.Sprintf(
			"\x1b[32m[DRIFT_SAVED]\x1b[0m Recorded %d state mutations (%d new hosts, %d port changes)",
			len(drifts), driftAnalysis.TotalNewHosts, driftAnalysis.TotalOpenedPorts,
		))
	}

	// 5. Finalize Metrics & Mark Complete
	totalDuration := time.Since(startTime)
	var openPortCount int
	for _, a := range stageCtx.LiveAssets {
		openPortCount += len(a.OpenPorts)
	}

	metrics := domain.ScanMetrics{
		TotalHosts:      len(stageCtx.DiscoveredHosts),
		LiveHosts:       len(stageCtx.LiveAssets),
		OpenPorts:       openPortCount,
		FindingsCount:   len(stageCtx.Findings),
		DurationSeconds: totalDuration.Seconds(),
		RequestsPerSec:  float64(len(stageCtx.DiscoveredHosts)*5) / totalDuration.Seconds(),
	}

	if o.db != nil && o.db.Pool != nil {
		_ = o.db.UpdateMetrics(scanCtx, job.ID, metrics)
		_ = o.db.UpdateCurrentStage(scanCtx, job.ID, domain.StageAISynthesis)
		_ = o.db.UpdateStatus(scanCtx, job.ID, domain.ScanCompleted, nil)
	}
	job.Status = domain.ScanCompleted

	o.emitLog(telemetryChan, job.ID, domain.StageCompleted, fmt.Sprintf(
		"\x1b[32;1m[SCAN_COMPLETE]\x1b[0m Target %s recon finished in %v: %d hosts, %d live, %d open ports, %d findings",
		scope.RootDomain, totalDuration.Round(time.Millisecond), metrics.TotalHosts, metrics.LiveHosts, metrics.OpenPorts, metrics.FindingsCount,
	))

	o.emitEvent(telemetryChan, domain.TelemetryEvent{
		Type:      "scan_finished",
		ScanJobID: job.ID,
		Stage:     domain.StageCompleted,
		Timestamp: time.Now(),
		Payload:   metrics,
	})

	return nil
}

func (o *PipelineOrchestrator) restoreStageState(ctx context.Context, stage domain.StageName, stageCtx *modules.StageContext) {
	switch stage {
	case domain.StageDiscovery, domain.StageProbing, domain.StagePortScan:
		assets, err := o.db.ListAssetsByScanJob(ctx, stageCtx.ScanJob.ID)
		if err == nil && len(assets) > 0 {
			stageCtx.LiveAssets = assets
			for _, a := range assets {
				stageCtx.DiscoveredHosts = append(stageCtx.DiscoveredHosts, a.FQDN)
			}
		}
	case domain.StageAuditor:
		findings, err := o.db.ListFindingsByScanJob(ctx, stageCtx.ScanJob.ID)
		if err == nil {
			stageCtx.Findings = findings
		}
	}
}

func (o *PipelineOrchestrator) emitLog(ch chan<- domain.TelemetryEvent, jobID uuid.UUID, stage domain.StageName, msg string) {
	ch <- domain.TelemetryEvent{
		Type:      "log",
		ScanJobID: jobID,
		Stage:     stage,
		Timestamp: time.Now(),
		Payload:   msg,
	}
}

func (o *PipelineOrchestrator) emitEvent(ch chan<- domain.TelemetryEvent, ev domain.TelemetryEvent) {
	ch <- ev
}

