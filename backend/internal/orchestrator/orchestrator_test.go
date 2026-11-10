package orchestrator

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"reconpilot/internal/config"
	"reconpilot/pkg/runner"
)

func TestOrchestratorWorkerPoolAndCancellation(t *testing.T) {
	cfg := &config.Config{
		Pipeline: config.PipelineConfig{
			MaxConcurrentScans: 2,
			GlobalScanTimeout:  10 * time.Second,
		},
	}

	orch := NewPipelineOrchestrator(cfg, nil, nil, runner.NewSafeRunner(5*time.Second, 128))

	id1 := uuid.New()
	id2 := uuid.New()

	// Verify initial scanning states
	if orch.IsScanning(id1) {
		t.Errorf("expected id1 to not be scanning")
	}

	// Register simulated active scan
	var cancelled bool
	orch.mu.Lock()
	orch.activeScans[id1] = func() {
		cancelled = true
	}
	orch.mu.Unlock()

	if !orch.IsScanning(id1) {
		t.Errorf("expected id1 to be actively scanning")
	}

	// Cancel scan
	success := orch.CancelScan(id1)
	if !success {
		t.Errorf("expected CancelScan to succeed for id1")
	}
	if !cancelled {
		t.Errorf("expected cancel func to be invoked")
	}
	if orch.IsScanning(id1) {
		t.Errorf("expected id1 to no longer be scanning after cancellation")
	}

	// Cancel non-existent scan
	if orch.CancelScan(id2) {
		t.Errorf("expected CancelScan to return false for non-existent id2")
	}

	// Verify worker pool semaphore sizing
	var wg sync.WaitGroup
	acquired := 0
	var mu sync.Mutex

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case orch.workerSem <- struct{}{}:
				mu.Lock()
				acquired++
				mu.Unlock()
			case <-time.After(100 * time.Millisecond):
			}
		}()
	}
	wg.Wait()

	if acquired != 2 {
		t.Errorf("expected to acquire 2 worker slots, got %d", acquired)
	}

	// Third acquire should block/fail within timeout
	select {
	case orch.workerSem <- struct{}{}:
		t.Errorf("expected third worker acquire to block due to MaxConcurrentScans=2")
		<-orch.workerSem
	default:
		// Succeeded in blocking
	}

	// Release slots
	<-orch.workerSem
	<-orch.workerSem
}

