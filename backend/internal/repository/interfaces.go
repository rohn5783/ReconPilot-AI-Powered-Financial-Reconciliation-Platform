package repository

import (
	"context"

	"github.com/google/uuid"

	"reconpilot/internal/domain"
)

// DiscrepancyFilter specifies optional criteria for querying discrepancies.
type DiscrepancyFilter struct {
	Level    *domain.DiscrepancyLevel    `json:"level,omitempty"`
	Type     *domain.DiscrepancyType     `json:"type,omitempty"`
	Severity *domain.DiscrepancySeverity `json:"severity,omitempty"`
	Status   *domain.DiscrepancyStatus   `json:"status,omitempty"`
	OrderID  *string                     `json:"order_id,omitempty"`
	PayoutID *string                     `json:"payout_id,omitempty"`
	Limit    int                         `json:"limit,omitempty"`
	Offset   int                         `json:"offset,omitempty"`
}

// OrderRepository handles persistence of internal merchant orders.
type OrderRepository interface {
	InsertBatch(ctx context.Context, orders []domain.Order) error
	FindAll(ctx context.Context) ([]domain.Order, error)
	FindByID(ctx context.Context, orderID string) (*domain.Order, error)
	Count(ctx context.Context) (int64, error)
	Clear(ctx context.Context) error
}

// PGRepository handles persistence of payment gateway settlements.
type PGRepository interface {
	InsertBatch(ctx context.Context, settlements []domain.PGSettlement) error
	FindAll(ctx context.Context) ([]domain.PGSettlement, error)
	FindByOrderID(ctx context.Context, orderID string) (*domain.PGSettlement, error)
	FindByPayoutID(ctx context.Context, payoutID string) ([]domain.PGSettlement, error)
	FindDistinctPayoutIDs(ctx context.Context) ([]string, error)
	Count(ctx context.Context) (int64, error)
	Clear(ctx context.Context) error
}

// BankRepository handles persistence of bank statement credit entries.
type BankRepository interface {
	InsertBatch(ctx context.Context, statements []domain.BankStatement) error
	FindAll(ctx context.Context) ([]domain.BankStatement, error)
	FindByUTR(ctx context.Context, utr string) (*domain.BankStatement, error)
	Count(ctx context.Context) (int64, error)
	Clear(ctx context.Context) error
}

// DiscrepancyRepository handles persistence of detected discrepancies and AI audit dossiers.
type DiscrepancyRepository interface {
	InsertBatch(ctx context.Context, discrepancies []domain.Discrepancy) error
	FindAll(ctx context.Context, filter DiscrepancyFilter) ([]domain.Discrepancy, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Discrepancy, error)
	FindByOrderID(ctx context.Context, orderID string) (*domain.Discrepancy, error)
	FindByPayoutID(ctx context.Context, payoutID string) (*domain.Discrepancy, error)
	UpdateDiagnosis(ctx context.Context, id uuid.UUID, diagnosis domain.AIDiagnosis) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DiscrepancyStatus) error
	Count(ctx context.Context) (int64, error)
	Clear(ctx context.Context) error
}

// UserRepository handles persistence and queries for authenticated platform operators.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	Count(ctx context.Context) (int64, error)
}

// UploadRepository handles audit logging, storage and retrieval of ingested files.
type UploadRepository interface {
	Create(ctx context.Context, record *domain.UploadRecord) error
	FindAll(ctx context.Context) ([]domain.UploadRecord, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.UploadRecord, error)
	Clear(ctx context.Context) error
}

// ReconciliationRepository aggregates all storage entities into a unified interface.
type ReconciliationRepository interface {
	Orders() OrderRepository
	PG() PGRepository
	Bank() BankRepository
	Discrepancies() DiscrepancyRepository
	Users() UserRepository
	Uploads() UploadRepository
	ResetAll(ctx context.Context) error
	Close() error
}

