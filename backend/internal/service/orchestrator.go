package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"reconpilot/internal/ai"
	"reconpilot/internal/domain"
	"reconpilot/internal/matching"
	"reconpilot/internal/repository"
	"reconpilot/internal/transport/stream"
	"reconpilot/pkg/money"
)

// ReconciliationService coordinates data ingestion, matching execution, AI audits, and state management.
type ReconciliationService struct {
	repo           repository.ReconciliationRepository
	matchingEngine *matching.Engine
	aiClient       ai.Diagnostician
	hub            *stream.Hub
	mu             sync.RWMutex
	lastResult     *matching.ReconciliationResult
}

// NewReconciliationService constructs the central reconciliation orchestrator.
func NewReconciliationService(
	repo repository.ReconciliationRepository,
	matchingEngine *matching.Engine,
	aiClient ai.Diagnostician,
	hub *stream.Hub,
) *ReconciliationService {
	svc := &ReconciliationService{
		repo:           repo,
		matchingEngine: matchingEngine,
		aiClient:       aiClient,
		hub:            hub,
	}

	// Prime initial state from repository
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _ = svc.RunReconciliation(ctx, nil, nil, nil)

	return svc
}

// Hub returns the stream broadcaster hub.
func (s *ReconciliationService) Hub() *stream.Hub {
	return s.hub
}

// RunReconciliation runs the complete 3-way matching and discrepancy diagnosis pipeline.
func (s *ReconciliationService) RunReconciliation(
	ctx context.Context,
	customOrders []domain.Order,
	customPG []domain.PGSettlement,
	customBank []domain.BankStatement,
) (*matching.ReconciliationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. If custom files were uploaded, persist them into repository
	if len(customOrders) > 0 {
		if err := s.repo.Orders().InsertBatch(ctx, customOrders); err != nil {
			return nil, fmt.Errorf("persist custom orders: %w", err)
		}
	}
	if len(customPG) > 0 {
		if err := s.repo.PG().InsertBatch(ctx, customPG); err != nil {
			return nil, fmt.Errorf("persist custom pg settlements: %w", err)
		}
	}
	if len(customBank) > 0 {
		if err := s.repo.Bank().InsertBatch(ctx, customBank); err != nil {
			return nil, fmt.Errorf("persist custom bank statements: %w", err)
		}
	}

	// 2. Fetch full working dataset from repository
	orders, err := s.repo.Orders().FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch orders: %w", err)
	}

	pgSettlements, err := s.repo.PG().FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch pg settlements: %w", err)
	}

	bankStatements, err := s.repo.Bank().FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch bank statements: %w", err)
	}

	// 3. Execute 3-way matching engine with real-time stream broadcast
	progressCallback := func(evt domain.PipelineTelemetryEvent) {
		if s.hub != nil {
			s.hub.Broadcast(evt)
		}
	}

	res, err := s.matchingEngine.Reconcile(ctx, orders, pgSettlements, bankStatements, progressCallback)
	if err != nil {
		return nil, fmt.Errorf("reconciliation engine failed: %w", err)
	}

	// 4. Clean prior discrepancies and persist newly detected ones
	_ = s.repo.Discrepancies().Clear(ctx)
	if len(res.Discrepancies) > 0 {
		if err := s.repo.Discrepancies().InsertBatch(ctx, res.Discrepancies); err != nil {
			return nil, fmt.Errorf("persist discrepancies: %w", err)
		}
	}

	// 5. Run AI diagnosis on discrepancies in the background or synchronous batch
	if s.aiClient != nil && len(res.Discrepancies) > 0 {
		if s.hub != nil {
			s.hub.Broadcast(domain.PipelineTelemetryEvent{
				Type:               "progress",
				Stage:              "AI_DIAGNOSIS",
				ProgressPercentage: 85.0,
				Message:            fmt.Sprintf("Synthesizing AI diagnoses for %d financial discrepancies...", len(res.Discrepancies)),
				Timestamp:          time.Now().UTC(),
			})
		}

		diagnosed, err := s.aiClient.DiagnoseBatch(ctx, res.Discrepancies)
		if err == nil {
			res.Discrepancies = diagnosed
			for _, d := range diagnosed {
				if d.AIDiagnosis != nil {
					_ = s.repo.Discrepancies().UpdateDiagnosis(ctx, d.ID, *d.AIDiagnosis)
				}
			}

			// Map updated AI diagnosis back onto trace items
			discMap := make(map[string]*domain.Discrepancy)
			for i := range res.Discrepancies {
				item := &res.Discrepancies[i]
				if item.OrderID != nil {
					discMap[*item.OrderID] = item
				}
			}
			for i := range res.TraceItems {
				if d, ok := discMap[res.TraceItems[i].OrderID]; ok {
					res.TraceItems[i].Discrepancy = d
				}
			}
		}
	}

	s.lastResult = res

	if s.hub != nil {
		s.hub.Broadcast(domain.PipelineTelemetryEvent{
			Type:               "completed",
			Stage:              "COMPLETED",
			ProgressPercentage: 100.0,
			Message:            fmt.Sprintf("Pipeline run finalized. Match Rate: %s%%", res.Summary.ThreeWayMatchRate.StringFixed(2)),
			Payload:            res.Summary,
			Timestamp:          time.Now().UTC(),
		})
	}

	return res, nil
}

// GetSummary returns executive workstation KPIs.
func (s *ReconciliationService) GetSummary(ctx context.Context) (*domain.ReconciliationSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.lastResult != nil {
		return &s.lastResult.Summary, nil
	}

	// If no cached run yet, compute empty summary
	return &domain.ReconciliationSummary{
		ThreeWayMatchRate: money.Zero(),
	}, nil
}

// GetTraces retrieves paginated, searchable 3-way trace items.
func (s *ReconciliationService) GetTraces(
	ctx context.Context,
	search string,
	statusFilter string,
	limit int,
	offset int,
) ([]domain.ThreeWayTraceItem, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.lastResult == nil {
		return []domain.ThreeWayTraceItem{}, 0, nil
	}

	var filtered []domain.ThreeWayTraceItem
	searchLower := strings.ToLower(strings.TrimSpace(search))

	for _, item := range s.lastResult.TraceItems {
		// Filter by status
		if statusFilter == "reconciled" && !item.IsReconciled {
			continue
		}
		if statusFilter == "discrepancy" && item.IsReconciled {
			continue
		}

		// Filter by search query
		if searchLower != "" {
			match := strings.Contains(strings.ToLower(item.OrderID), searchLower)
			if !match && item.PGTxnID != nil {
				match = strings.Contains(strings.ToLower(*item.PGTxnID), searchLower)
			}
			if !match && item.BankUTR != nil {
				match = strings.Contains(strings.ToLower(*item.BankUTR), searchLower)
			}
			if !match {
				continue
			}
		}

		filtered = append(filtered, item)
	}

	total := len(filtered)
	if offset >= total {
		return []domain.ThreeWayTraceItem{}, total, nil
	}

	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}

	return filtered[offset:end], total, nil
}

// GetPayoutBatches retrieves Level-2 payout batch groups.
func (s *ReconciliationService) GetPayoutBatches(ctx context.Context) ([]domain.PayoutBatchGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.lastResult == nil {
		return []domain.PayoutBatchGroup{}, nil
	}
	return s.lastResult.PayoutBatches, nil
}

// GetDiscrepancies returns discrepancies filtered by level, severity, or status.
func (s *ReconciliationService) GetDiscrepancies(ctx context.Context, filter repository.DiscrepancyFilter) ([]domain.Discrepancy, error) {
	return s.repo.Discrepancies().FindAll(ctx, filter)
}

// GetDiscrepancyByID retrieves a single discrepancy audit dossier.
func (s *ReconciliationService) GetDiscrepancyByID(ctx context.Context, id uuid.UUID) (*domain.Discrepancy, error) {
	return s.repo.Discrepancies().FindByID(ctx, id)
}

// DiagnoseDiscrepancy triggers an on-demand Gemini AI analysis for a specific discrepancy.
func (s *ReconciliationService) DiagnoseDiscrepancy(ctx context.Context, id uuid.UUID) (*domain.AIDiagnosis, error) {
	d, err := s.repo.Discrepancies().FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find discrepancy: %w", err)
	}
	if d == nil {
		return nil, fmt.Errorf("discrepancy %s not found", id)
	}

	diag, err := s.aiClient.Diagnose(ctx, d)
	if err != nil {
		return nil, fmt.Errorf("diagnose discrepancy: %w", err)
	}

	if err := s.repo.Discrepancies().UpdateDiagnosis(ctx, id, *diag); err != nil {
		return nil, fmt.Errorf("save diagnosis: %w", err)
	}

	// Update cached copy
	s.mu.Lock()
	if s.lastResult != nil {
		for i := range s.lastResult.Discrepancies {
			if s.lastResult.Discrepancies[i].ID == id {
				s.lastResult.Discrepancies[i].AIDiagnosis = diag
				break
			}
		}
	}
	s.mu.Unlock()

	return diag, nil
}

// UpdateDiscrepancyStatus updates the status of a discrepancy (e.g. IN_DISPUTE, RESOLVED).
func (s *ReconciliationService) UpdateDiscrepancyStatus(ctx context.Context, id uuid.UUID, status domain.DiscrepancyStatus) error {
	if err := s.repo.Discrepancies().UpdateStatus(ctx, id, status); err != nil {
		return fmt.Errorf("update status in repository: %w", err)
	}

	s.mu.Lock()
	if s.lastResult != nil {
		for i := range s.lastResult.Discrepancies {
			if s.lastResult.Discrepancies[i].ID == id {
				s.lastResult.Discrepancies[i].Status = status
				break
			}
		}
	}
	s.mu.Unlock()

	return nil
}

// SeedMockData clears and re-populates the repository with the rich mock dataset and re-runs reconciliation.
func (s *ReconciliationService) SeedMockData(ctx context.Context) error {
	if err := s.repo.ResetAll(ctx); err != nil {
		return fmt.Errorf("reset repository: %w", err)
	}

	// If mock repo, call SeedRealisticData
	if mockRepo, ok := s.repo.(*repository.MockRepository); ok {
		mockRepo.SeedRealisticData()
	}

	_, err := s.RunReconciliation(ctx, nil, nil, nil)
	return err
}

// LastResult returns the current reconciliation state.
func (s *ReconciliationService) LastResult() *matching.ReconciliationResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastResult
}
