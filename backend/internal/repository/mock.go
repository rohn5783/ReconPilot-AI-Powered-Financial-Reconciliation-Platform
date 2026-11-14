package repository

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"

	"reconpilot/internal/domain"
)

// MockRepository provides a thread-safe in-memory database store.
// It contains zero hardcoded data and stores only dynamically ingested/inserted financial records.
type MockRepository struct {
	mu            sync.RWMutex
	orders        *MockOrderRepo
	pg            *MockPGRepo
	bank          *MockBankRepo
	discrepancies *MockDiscrepancyRepo
	users         *MockUserRepo
}

// NewMockRepository constructs a clean, empty in-memory repository.
func NewMockRepository(seedData bool) *MockRepository {
	orders := &MockOrderRepo{items: make(map[string]domain.Order)}
	pg := &MockPGRepo{items: make(map[string]domain.PGSettlement)}
	bank := &MockBankRepo{items: make(map[string]domain.BankStatement)}
	discrepancies := &MockDiscrepancyRepo{items: make(map[uuid.UUID]domain.Discrepancy)}
	users := &MockUserRepo{
		usersByMail: make(map[string]*domain.User),
		usersByID:   make(map[uuid.UUID]*domain.User),
	}

	repo := &MockRepository{
		orders:        orders,
		pg:            pg,
		bank:          bank,
		discrepancies: discrepancies,
		users:         users,
	}

	return repo
}

func (r *MockRepository) Orders() OrderRepository             { return r.orders }
func (r *MockRepository) PG() PGRepository                     { return r.pg }
func (r *MockRepository) Bank() BankRepository                 { return r.bank }
func (r *MockRepository) Discrepancies() DiscrepancyRepository { return r.discrepancies }
func (r *MockRepository) Users() UserRepository                 { return r.users }

func (r *MockRepository) ResetAll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_ = r.orders.Clear(ctx)
	_ = r.pg.Clear(ctx)
	_ = r.bank.Clear(ctx)
	_ = r.discrepancies.Clear(ctx)
	return nil
}

func (r *MockRepository) Close() error {
	return nil
}

// SeedRealisticData is a clean no-op; all hardcoded fake records have been eliminated.
func (r *MockRepository) SeedRealisticData() {}

// MockOrderRepo implements OrderRepository in-memory.
type MockOrderRepo struct {
	mu    sync.RWMutex
	items map[string]domain.Order
}

func (r *MockOrderRepo) InsertBatch(ctx context.Context, orders []domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, o := range orders {
		r.items[o.OrderID] = o
	}
	return nil
}

func (r *MockOrderRepo) FindAll(ctx context.Context) ([]domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]domain.Order, 0, len(r.items))
	for _, o := range r.items {
		res = append(res, o)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].CreatedAt.After(res[j].CreatedAt)
	})
	return res, nil
}

func (r *MockOrderRepo) FindByID(ctx context.Context, orderID string) (*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, exists := r.items[orderID]
	if !exists {
		return nil, nil
	}
	return &o, nil
}

func (r *MockOrderRepo) Count(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.items)), nil
}

func (r *MockOrderRepo) Clear(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = make(map[string]domain.Order)
	return nil
}

// MockPGRepo implements PGRepository in-memory.
type MockPGRepo struct {
	mu    sync.RWMutex
	items map[string]domain.PGSettlement
}

func (r *MockPGRepo) InsertBatch(ctx context.Context, settlements []domain.PGSettlement) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range settlements {
		r.items[s.PGTxnID] = s
	}
	return nil
}

func (r *MockPGRepo) FindAll(ctx context.Context) ([]domain.PGSettlement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]domain.PGSettlement, 0, len(r.items))
	for _, s := range r.items {
		res = append(res, s)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].SettledAt.After(res[j].SettledAt)
	})
	return res, nil
}

func (r *MockPGRepo) FindByOrderID(ctx context.Context, orderID string) (*domain.PGSettlement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.items {
		if s.OrderID == orderID {
			return &s, nil
		}
	}
	return nil, nil
}

func (r *MockPGRepo) FindByPayoutID(ctx context.Context, payoutID string) ([]domain.PGSettlement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var res []domain.PGSettlement
	for _, s := range r.items {
		if s.PayoutID == payoutID {
			res = append(res, s)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].SettledAt.After(res[j].SettledAt)
	})
	return res, nil
}

func (r *MockPGRepo) FindDistinctPayoutIDs(ctx context.Context) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[string]bool)
	var res []string
	for _, s := range r.items {
		if s.PayoutID != "" && !seen[s.PayoutID] {
			seen[s.PayoutID] = true
			res = append(res, s.PayoutID)
		}
	}
	sort.Strings(res)
	return res, nil
}

func (r *MockPGRepo) Count(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.items)), nil
}

func (r *MockPGRepo) Clear(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = make(map[string]domain.PGSettlement)
	return nil
}

// MockBankRepo implements BankRepository in-memory.
type MockBankRepo struct {
	mu    sync.RWMutex
	items map[string]domain.BankStatement
}

func (r *MockBankRepo) InsertBatch(ctx context.Context, statements []domain.BankStatement) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range statements {
		r.items[b.BankTxnID] = b
	}
	return nil
}

func (r *MockBankRepo) FindAll(ctx context.Context) ([]domain.BankStatement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]domain.BankStatement, 0, len(r.items))
	for _, b := range r.items {
		res = append(res, b)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].TransactionDate.After(res[j].TransactionDate)
	})
	return res, nil
}

func (r *MockBankRepo) FindByUTR(ctx context.Context, utr string) (*domain.BankStatement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, b := range r.items {
		if strings.EqualFold(b.UTRNumber, utr) {
			return &b, nil
		}
	}
	return nil, nil
}

func (r *MockBankRepo) Count(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.items)), nil
}

func (r *MockBankRepo) Clear(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = make(map[string]domain.BankStatement)
	return nil
}

// MockDiscrepancyRepo implements DiscrepancyRepository in-memory.
type MockDiscrepancyRepo struct {
	mu    sync.RWMutex
	items map[uuid.UUID]domain.Discrepancy
}

func (r *MockDiscrepancyRepo) InsertBatch(ctx context.Context, discrepancies []domain.Discrepancy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, d := range discrepancies {
		r.items[d.ID] = d
	}
	return nil
}

func (r *MockDiscrepancyRepo) FindAll(ctx context.Context, filter DiscrepancyFilter) ([]domain.Discrepancy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []domain.Discrepancy
	for _, d := range r.items {
		if filter.Level != nil && d.DiscrepancyLevel != *filter.Level {
			continue
		}
		if filter.Type != nil && d.DiscrepancyType != *filter.Type {
			continue
		}
		if filter.Severity != nil && d.Severity != *filter.Severity {
			continue
		}
		if filter.Status != nil && d.Status != *filter.Status {
			continue
		}
		if filter.OrderID != nil && (d.OrderID == nil || *d.OrderID != *filter.OrderID) {
			continue
		}
		if filter.PayoutID != nil && (d.PayoutID == nil || *d.PayoutID != *filter.PayoutID) {
			continue
		}
		matched = append(matched, d)
	}

	sort.Slice(matched, func(i, j int) bool {
		return matched[i].DetectedAt.After(matched[j].DetectedAt)
	})

	if filter.Offset > 0 {
		if filter.Offset >= len(matched) {
			return []domain.Discrepancy{}, nil
		}
		matched = matched[filter.Offset:]
	}

	if filter.Limit > 0 && len(matched) > filter.Limit {
		matched = matched[:filter.Limit]
	}

	return matched, nil
}

func (r *MockDiscrepancyRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Discrepancy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, exists := r.items[id]
	if !exists {
		return nil, nil
	}
	return &d, nil
}

func (r *MockDiscrepancyRepo) FindByOrderID(ctx context.Context, orderID string) (*domain.Discrepancy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, d := range r.items {
		if d.OrderID != nil && *d.OrderID == orderID {
			return &d, nil
		}
	}
	return nil, nil
}

func (r *MockDiscrepancyRepo) FindByPayoutID(ctx context.Context, payoutID string) (*domain.Discrepancy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, d := range r.items {
		if d.PayoutID != nil && *d.PayoutID == payoutID {
			return &d, nil
		}
	}
	return nil, nil
}

func (r *MockDiscrepancyRepo) UpdateDiagnosis(ctx context.Context, id uuid.UUID, diagnosis domain.AIDiagnosis) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, exists := r.items[id]
	if !exists {
		return errors.New("discrepancy not found")
	}
	d.AIDiagnosis = &diagnosis
	r.items[id] = d
	return nil
}

func (r *MockDiscrepancyRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DiscrepancyStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, exists := r.items[id]
	if !exists {
		return errors.New("discrepancy not found")
	}
	d.Status = status
	r.items[id] = d
	return nil
}

func (r *MockDiscrepancyRepo) Count(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.items)), nil
}

func (r *MockDiscrepancyRepo) Clear(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = make(map[uuid.UUID]domain.Discrepancy)
	return nil
}

// MockUserRepo provides thread-safe in-memory user storage.
type MockUserRepo struct {
	mu          sync.RWMutex
	usersByMail map[string]*domain.User
	usersByID   map[uuid.UUID]*domain.User
}

func (r *MockUserRepo) Create(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	emailKey := strings.ToLower(strings.TrimSpace(user.Email))
	if _, exists := r.usersByMail[emailKey]; exists {
		return errors.New("user with this email already exists")
	}

	r.usersByMail[emailKey] = user
	r.usersByID[user.ID] = user
	return nil
}

func (r *MockUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	emailKey := strings.ToLower(strings.TrimSpace(email))
	u, ok := r.usersByMail[emailKey]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (r *MockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, ok := r.usersByID[id]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (r *MockUserRepo) Count(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.usersByID)), nil
}
