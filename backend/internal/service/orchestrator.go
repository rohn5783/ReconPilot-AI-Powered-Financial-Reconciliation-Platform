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
	"reconpilot/internal/seed"
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

// ResetData completely wipes all orders, settlements, bank statements, and discrepancies from the repository,
// resets the internal state to empty, and broadcasts a telemetry event.
func (s *ReconciliationService) ResetData(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.repo.Discrepancies().Clear(ctx); err != nil {
		return fmt.Errorf("clear discrepancies: %w", err)
	}
	if err := s.repo.Bank().Clear(ctx); err != nil {
		return fmt.Errorf("clear bank statements: %w", err)
	}
	if err := s.repo.PG().Clear(ctx); err != nil {
		return fmt.Errorf("clear pg settlements: %w", err)
	}
	if err := s.repo.Orders().Clear(ctx); err != nil {
		return fmt.Errorf("clear orders: %w", err)
	}

	emptySummary := domain.ReconciliationSummary{
		TotalOrders:              0,
		TotalGrossOrderVolume:    money.Zero(),
		TotalPGSettlements:       0,
		TotalPGFees:              money.Zero(),
		TotalPGTax:               money.Zero(),
		ExpectedNetPayout:        money.Zero(),
		TotalBankCredits:         money.Zero(),
		NetDiscrepancyVariance:   money.Zero(),
		Level1DiscrepanciesCount: 0,
		Level2DiscrepanciesCount: 0,
		ThreeWayMatchRate:        money.FromStringOrZero("100"),
		ReconciledOrdersCount:    0,
		UnreconciledOrdersCount:  0,
	}

	s.lastResult = &matching.ReconciliationResult{
		Discrepancies: []domain.Discrepancy{},
		TraceItems:    []domain.ThreeWayTraceItem{},
		PayoutBatches: []domain.PayoutBatchGroup{},
		Summary:       emptySummary,
	}

	if s.hub != nil {
		s.hub.Broadcast(domain.PipelineTelemetryEvent{
			Type:               "completed",
			Stage:              "COMPLETED",
			ProgressPercentage: 100.0,
			Message:            "All financial ledger data erased successfully. Ledger reset to zero.",
			Payload:            emptySummary,
			Timestamp:          time.Now().UTC(),
		})
	}

	return nil
}

// SeedMockData clears and re-populates the repository with the rich mock dataset and re-runs reconciliation.
func (s *ReconciliationService) SeedMockData(ctx context.Context) error {
	if err := seed.SeedRepository(ctx, s.repo); err != nil {
		return fmt.Errorf("seed repository: %w", err)
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

// RecordUpload records an uploaded file audit entry into storage.
func (s *ReconciliationService) RecordUpload(ctx context.Context, record *domain.UploadRecord) error {
	return s.repo.Uploads().Create(ctx, record)
}

// GetUploads retrieves all historical upload and ingestion records.
func (s *ReconciliationService) GetUploads(ctx context.Context) ([]domain.UploadRecord, error) {
	return s.repo.Uploads().FindAll(ctx)
}

// GetUploadByID retrieves a single upload record by ID.
func (s *ReconciliationService) GetUploadByID(ctx context.Context, id uuid.UUID) (*domain.UploadRecord, error) {
	return s.repo.Uploads().FindByID(ctx, id)
}

// GetContractRules retrieves all contract rate and tolerance configurations.
func (s *ReconciliationService) GetContractRules(ctx context.Context) ([]domain.ContractRule, error) {
	rules, err := s.repo.ContractRules().FindAll(ctx)
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		defaultRule := domain.ContractRule{
			ID:                uuid.New(),
			Name:              "Institutional Base Contract (Razorpay / Stripe)",
			MDRRate:           money.FromStringOrZero("2.0000"),
			GSTRate:           money.FromStringOrZero("18.0000"),
			FeeTolerance:      money.FromStringOrZero("0.0100"),
			RoundingPrecision: 4,
			RoundingMode:      "HALF_EVEN",
			IsActive:          true,
			Description:       "Standard 2.0% MDR with 18% statutory GST and ₹0.01 fee tolerance (Banker's rounding)",
			CreatedAt:         time.Now().UTC(),
			UpdatedAt:         time.Now().UTC(),
		}
		_ = s.repo.ContractRules().Create(ctx, &defaultRule)
		rules = []domain.ContractRule{defaultRule}
	}
	return rules, nil
}

// CreateContractRule saves a new user-defined contract configuration.
func (s *ReconciliationService) CreateContractRule(ctx context.Context, rule *domain.ContractRule) error {
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}
	if err := s.repo.ContractRules().Create(ctx, rule); err != nil {
		return err
	}
	if rule.IsActive {
		s.applyContractRuleToEngine(ctx, rule)
	}
	return nil
}

// UpdateContractRule updates an existing contract rule and refreshes reconciliation if active.
func (s *ReconciliationService) UpdateContractRule(ctx context.Context, rule *domain.ContractRule) error {
	if err := s.repo.ContractRules().Update(ctx, rule); err != nil {
		return err
	}
	if rule.IsActive {
		s.applyContractRuleToEngine(ctx, rule)
	}
	return nil
}

// ActivateContractRule marks a contract rule as active and updates the matching engine rates.
func (s *ReconciliationService) ActivateContractRule(ctx context.Context, id uuid.UUID) (*domain.ContractRule, error) {
	if err := s.repo.ContractRules().SetActive(ctx, id); err != nil {
		return nil, err
	}
	rule, err := s.repo.ContractRules().FindByID(ctx, id)
	if err != nil || rule == nil {
		return nil, fmt.Errorf("lookup activated rule: %w", err)
	}
	s.applyContractRuleToEngine(ctx, rule)
	return rule, nil
}

// DeleteContractRule deletes an inactive rule.
func (s *ReconciliationService) DeleteContractRule(ctx context.Context, id uuid.UUID) error {
	return s.repo.ContractRules().Delete(ctx, id)
}

func (s *ReconciliationService) applyContractRuleToEngine(ctx context.Context, rule *domain.ContractRule) {
	currentCfg := s.matchingEngine.Config()
	s.matchingEngine.UpdateConfig(matching.EngineConfig{
		DefaultFeeRate: rule.MDRRate,
		DefaultTaxRate: rule.GSTRate,
		FeeTolerance:   rule.FeeTolerance,
		MaxConcurrency: currentCfg.MaxConcurrency,
	})
	// Trigger re-reconciliation with the new rates
	_, _ = s.RunReconciliation(ctx, nil, nil, nil)
	if s.hub != nil {
		s.hub.Broadcast(domain.PipelineTelemetryEvent{
			Type:               "kpi_update",
			Stage:              "COMPLETED",
			ProgressPercentage: 100.0,
			Message:            fmt.Sprintf("Contract Rule %q applied: MDR %s%%, GST %s%%, Tolerance %s", rule.Name, rule.MDRRate.StringFixed(2), rule.GSTRate.StringFixed(2), rule.FeeTolerance.StringFixed(4)),
			Timestamp:          time.Now().UTC(),
		})
	}
}
