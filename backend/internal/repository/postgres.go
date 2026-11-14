package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"reconpilot/internal/domain"
)

// PostgresRepository implements ReconciliationRepository over PostgreSQL.
type PostgresRepository struct {
	pool          *pgxpool.Pool
	orders        *PostgresOrderRepo
	pg            *PostgresPGRepo
	bank          *PostgresBankRepo
	discrepancies *PostgresDiscrepancyRepo
	users         *PostgresUserRepo
}

// NewPostgresRepository initializes a connection pool and returns a PostgresRepository.
func NewPostgresRepository(ctx context.Context, databaseURL string) (*PostgresRepository, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres database url: %w", err)
	}

	cfg.MaxConns = 25
	cfg.MinConns = 5
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	repo := &PostgresRepository{
		pool:          pool,
		orders:        &PostgresOrderRepo{pool: pool},
		pg:            &PostgresPGRepo{pool: pool},
		bank:          &PostgresBankRepo{pool: pool},
		discrepancies: &PostgresDiscrepancyRepo{pool: pool},
		users:         &PostgresUserRepo{pool: pool},
	}

	if err := repo.AutoMigrate(ctx); err != nil {
		return nil, fmt.Errorf("auto migrate postgres schema: %w", err)
	}

	return repo, nil
}

func (r *PostgresRepository) AutoMigrate(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, schemaDDL)
	return err
}

const schemaDDL = `
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS orders (
    order_id VARCHAR(128) PRIMARY KEY,
    customer_id VARCHAR(128) NOT NULL,
    gross_amount NUMERIC(18, 4) NOT NULL CHECK (gross_amount >= 0),
    currency VARCHAR(10) NOT NULL DEFAULT 'INR',
    order_status VARCHAR(64) NOT NULL CHECK (order_status IN ('CAPTURED', 'PENDING', 'FAILED', 'REFUNDED')),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(order_status);

CREATE TABLE IF NOT EXISTS pg_settlements (
    pg_txn_id VARCHAR(128) PRIMARY KEY,
    order_id VARCHAR(128) NOT NULL,
    gross_amount NUMERIC(18, 4) NOT NULL CHECK (gross_amount >= 0),
    pg_fee NUMERIC(18, 4) NOT NULL CHECK (pg_fee >= 0),
    tax NUMERIC(18, 4) NOT NULL CHECK (tax >= 0),
    net_amount NUMERIC(18, 4) NOT NULL,
    payout_id VARCHAR(128) NOT NULL,
    payout_status VARCHAR(64) NOT NULL CHECK (payout_status IN ('SETTLED', 'PROCESSED', 'PENDING', 'FAILED')),
    settled_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_pg_settlements_order_id ON pg_settlements(order_id);
CREATE INDEX IF NOT EXISTS idx_pg_settlements_payout_id ON pg_settlements(payout_id);
CREATE INDEX IF NOT EXISTS idx_pg_settlements_settled_at ON pg_settlements(settled_at DESC);
CREATE INDEX IF NOT EXISTS idx_pg_settlements_payout_status ON pg_settlements(payout_status);

CREATE TABLE IF NOT EXISTS bank_statements (
    bank_txn_id VARCHAR(128) PRIMARY KEY,
    utr_number VARCHAR(128) UNIQUE NOT NULL,
    credit_amount NUMERIC(18, 4) NOT NULL CHECK (credit_amount >= 0),
    description TEXT NOT NULL,
    transaction_date TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_bank_statements_utr ON bank_statements(utr_number);
CREATE INDEX IF NOT EXISTS idx_bank_statements_txn_date ON bank_statements(transaction_date DESC);

CREATE TABLE IF NOT EXISTS discrepancies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id VARCHAR(128) NULL,
    payout_id VARCHAR(128) NULL,
    discrepancy_level VARCHAR(32) NOT NULL CHECK (discrepancy_level IN ('LEVEL_1_ORDER_PG', 'LEVEL_2_PG_BANK')),
    discrepancy_type VARCHAR(64) NOT NULL CHECK (discrepancy_type IN (
        'FEE_OVERCHARGE',
        'TAX_CALCULATION_ERROR',
        'ORDER_MISSING_IN_PG',
        'PG_ORPHAN_TXN',
        'CAPTURE_STATUS_MISMATCH',
        'GROSS_AMOUNT_MISMATCH',
        'UNSETTLED_PAYOUT',
        'BANK_MISSING',
        'PAYOUT_SHORTFALL',
        'PAYOUT_SURPLUS'
    )),
    expected_amount NUMERIC(18, 4) NOT NULL,
    actual_amount NUMERIC(18, 4) NOT NULL,
    mismatch_amount NUMERIC(18, 4) NOT NULL,
    severity VARCHAR(32) NOT NULL CHECK (severity IN ('CRITICAL', 'HIGH', 'MEDIUM', 'LOW')),
    raw_evidence JSONB NOT NULL,
    ai_diagnosis JSONB NULL,
    status VARCHAR(64) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'IN_DISPUTE', 'RESOLVED', 'WRITTEN_OFF')),
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_discrepancies_order_id ON discrepancies(order_id);
CREATE INDEX IF NOT EXISTS idx_discrepancies_payout_id ON discrepancies(payout_id);
CREATE INDEX IF NOT EXISTS idx_discrepancies_level_type ON discrepancies(discrepancy_level, discrepancy_type);
CREATE INDEX IF NOT EXISTS idx_discrepancies_status ON discrepancies(status);
CREATE INDEX IF NOT EXISTS idx_discrepancies_severity ON discrepancies(severity);
CREATE INDEX IF NOT EXISTS idx_discrepancies_detected_at ON discrepancies(detected_at DESC);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    company_name VARCHAR(255) NOT NULL DEFAULT '',
    role VARCHAR(50) NOT NULL DEFAULT 'FINANCE_ANALYST',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
`

func (r *PostgresRepository) Orders() OrderRepository             { return r.orders }
func (r *PostgresRepository) PG() PGRepository                     { return r.pg }
func (r *PostgresRepository) Bank() BankRepository                 { return r.bank }
func (r *PostgresRepository) Discrepancies() DiscrepancyRepository { return r.discrepancies }
func (r *PostgresRepository) Users() UserRepository                 { return r.users }

func (r *PostgresRepository) ResetAll(ctx context.Context) error {
	queries := []string{
		"TRUNCATE TABLE discrepancies CASCADE;",
		"TRUNCATE TABLE bank_statements CASCADE;",
		"TRUNCATE TABLE pg_settlements CASCADE;",
		"TRUNCATE TABLE orders CASCADE;",
	}
	for _, q := range queries {
		if _, err := r.pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("reset table exec %q: %w", q, err)
		}
	}
	return nil
}

func (r *PostgresRepository) Close() error {
	r.pool.Close()
	return nil
}

// PostgresOrderRepo provides PostgreSQL persistence for orders.
type PostgresOrderRepo struct {
	pool *pgxpool.Pool
}

func (r *PostgresOrderRepo) InsertBatch(ctx context.Context, orders []domain.Order) error {
	if len(orders) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO orders (order_id, customer_id, gross_amount, currency, order_status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (order_id) DO UPDATE SET
			customer_id = EXCLUDED.customer_id,
			gross_amount = EXCLUDED.gross_amount,
			currency = EXCLUDED.currency,
			order_status = EXCLUDED.order_status,
			created_at = EXCLUDED.created_at;
	`
	for _, o := range orders {
		batch.Queue(query, o.OrderID, o.CustomerID, o.GrossAmount, o.Currency, string(o.OrderStatus), o.CreatedAt)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range orders {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert order batch exec: %w", err)
		}
	}
	return nil
}

func (r *PostgresOrderRepo) FindAll(ctx context.Context) ([]domain.Order, error) {
	query := `SELECT order_id, customer_id, gross_amount, currency, order_status, created_at FROM orders ORDER BY created_at DESC;`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		var statusStr string
		if err := rows.Scan(&o.OrderID, &o.CustomerID, &o.GrossAmount, &o.Currency, &statusStr, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		o.OrderStatus = domain.OrderStatus(statusStr)
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (r *PostgresOrderRepo) FindByID(ctx context.Context, orderID string) (*domain.Order, error) {
	query := `SELECT order_id, customer_id, gross_amount, currency, order_status, created_at FROM orders WHERE order_id = $1;`
	var o domain.Order
	var statusStr string
	err := r.pool.QueryRow(ctx, query, orderID).Scan(&o.OrderID, &o.CustomerID, &o.GrossAmount, &o.Currency, &statusStr, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find order by id: %w", err)
	}
	o.OrderStatus = domain.OrderStatus(statusStr)
	return &o, nil
}

func (r *PostgresOrderRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM orders;").Scan(&count)
	return count, err
}

func (r *PostgresOrderRepo) Clear(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, "TRUNCATE TABLE orders CASCADE;")
	return err
}

// PostgresPGRepo provides PostgreSQL persistence for PG settlements.
type PostgresPGRepo struct {
	pool *pgxpool.Pool
}

func (r *PostgresPGRepo) InsertBatch(ctx context.Context, settlements []domain.PGSettlement) error {
	if len(settlements) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO pg_settlements (pg_txn_id, order_id, gross_amount, pg_fee, tax, net_amount, payout_id, payout_status, settled_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (pg_txn_id) DO UPDATE SET
			order_id = EXCLUDED.order_id,
			gross_amount = EXCLUDED.gross_amount,
			pg_fee = EXCLUDED.pg_fee,
			tax = EXCLUDED.tax,
			net_amount = EXCLUDED.net_amount,
			payout_id = EXCLUDED.payout_id,
			payout_status = EXCLUDED.payout_status,
			settled_at = EXCLUDED.settled_at;
	`
	for _, s := range settlements {
		batch.Queue(query, s.PGTxnID, s.OrderID, s.GrossAmount, s.PGFee, s.Tax, s.NetAmount, s.PayoutID, string(s.PayoutStatus), s.SettledAt)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range settlements {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert pg_settlement batch exec: %w", err)
		}
	}
	return nil
}

func (r *PostgresPGRepo) FindAll(ctx context.Context) ([]domain.PGSettlement, error) {
	query := `SELECT pg_txn_id, order_id, gross_amount, pg_fee, tax, net_amount, payout_id, payout_status, settled_at FROM pg_settlements ORDER BY settled_at DESC;`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query pg_settlements: %w", err)
	}
	defer rows.Close()

	var settlements []domain.PGSettlement
	for rows.Next() {
		var s domain.PGSettlement
		var statusStr string
		if err := rows.Scan(&s.PGTxnID, &s.OrderID, &s.GrossAmount, &s.PGFee, &s.Tax, &s.NetAmount, &s.PayoutID, &statusStr, &s.SettledAt); err != nil {
			return nil, fmt.Errorf("scan pg_settlement: %w", err)
		}
		s.PayoutStatus = domain.PayoutStatus(statusStr)
		settlements = append(settlements, s)
	}
	return settlements, rows.Err()
}

func (r *PostgresPGRepo) FindByOrderID(ctx context.Context, orderID string) (*domain.PGSettlement, error) {
	query := `SELECT pg_txn_id, order_id, gross_amount, pg_fee, tax, net_amount, payout_id, payout_status, settled_at FROM pg_settlements WHERE order_id = $1;`
	var s domain.PGSettlement
	var statusStr string
	err := r.pool.QueryRow(ctx, query, orderID).Scan(&s.PGTxnID, &s.OrderID, &s.GrossAmount, &s.PGFee, &s.Tax, &s.NetAmount, &s.PayoutID, &statusStr, &s.SettledAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find pg_settlement by order_id: %w", err)
	}
	s.PayoutStatus = domain.PayoutStatus(statusStr)
	return &s, nil
}

func (r *PostgresPGRepo) FindByPayoutID(ctx context.Context, payoutID string) ([]domain.PGSettlement, error) {
	query := `SELECT pg_txn_id, order_id, gross_amount, pg_fee, tax, net_amount, payout_id, payout_status, settled_at FROM pg_settlements WHERE payout_id = $1 ORDER BY settled_at DESC;`
	rows, err := r.pool.Query(ctx, query, payoutID)
	if err != nil {
		return nil, fmt.Errorf("find pg_settlements by payout_id: %w", err)
	}
	defer rows.Close()

	var settlements []domain.PGSettlement
	for rows.Next() {
		var s domain.PGSettlement
		var statusStr string
		if err := rows.Scan(&s.PGTxnID, &s.OrderID, &s.GrossAmount, &s.PGFee, &s.Tax, &s.NetAmount, &s.PayoutID, &statusStr, &s.SettledAt); err != nil {
			return nil, fmt.Errorf("scan pg_settlement: %w", err)
		}
		s.PayoutStatus = domain.PayoutStatus(statusStr)
		settlements = append(settlements, s)
	}
	return settlements, rows.Err()
}

func (r *PostgresPGRepo) FindDistinctPayoutIDs(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT payout_id FROM pg_settlements WHERE payout_id <> '' ORDER BY payout_id ASC;`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query distinct payout ids: %w", err)
	}
	defer rows.Close()

	var payoutIDs []string
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			return nil, fmt.Errorf("scan payout_id: %w", err)
		}
		payoutIDs = append(payoutIDs, pid)
	}
	return payoutIDs, rows.Err()
}

func (r *PostgresPGRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM pg_settlements;").Scan(&count)
	return count, err
}

func (r *PostgresPGRepo) Clear(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, "TRUNCATE TABLE pg_settlements CASCADE;")
	return err
}

// PostgresBankRepo provides PostgreSQL persistence for bank statement records.
type PostgresBankRepo struct {
	pool *pgxpool.Pool
}

func (r *PostgresBankRepo) InsertBatch(ctx context.Context, statements []domain.BankStatement) error {
	if len(statements) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO bank_statements (bank_txn_id, utr_number, credit_amount, description, transaction_date)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (bank_txn_id) DO UPDATE SET
			utr_number = EXCLUDED.utr_number,
			credit_amount = EXCLUDED.credit_amount,
			description = EXCLUDED.description,
			transaction_date = EXCLUDED.transaction_date;
	`
	for _, b := range statements {
		batch.Queue(query, b.BankTxnID, b.UTRNumber, b.CreditAmount, b.Description, b.TransactionDate)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range statements {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert bank_statement batch exec: %w", err)
		}
	}
	return nil
}

func (r *PostgresBankRepo) FindAll(ctx context.Context) ([]domain.BankStatement, error) {
	query := `SELECT bank_txn_id, utr_number, credit_amount, description, transaction_date FROM bank_statements ORDER BY transaction_date DESC;`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query bank_statements: %w", err)
	}
	defer rows.Close()

	var statements []domain.BankStatement
	for rows.Next() {
		var b domain.BankStatement
		if err := rows.Scan(&b.BankTxnID, &b.UTRNumber, &b.CreditAmount, &b.Description, &b.TransactionDate); err != nil {
			return nil, fmt.Errorf("scan bank_statement: %w", err)
		}
		statements = append(statements, b)
	}
	return statements, rows.Err()
}

func (r *PostgresBankRepo) FindByUTR(ctx context.Context, utr string) (*domain.BankStatement, error) {
	query := `SELECT bank_txn_id, utr_number, credit_amount, description, transaction_date FROM bank_statements WHERE utr_number = $1;`
	var b domain.BankStatement
	err := r.pool.QueryRow(ctx, query, utr).Scan(&b.BankTxnID, &b.UTRNumber, &b.CreditAmount, &b.Description, &b.TransactionDate)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find bank_statement by utr: %w", err)
	}
	return &b, nil
}

func (r *PostgresBankRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM bank_statements;").Scan(&count)
	return count, err
}

func (r *PostgresBankRepo) Clear(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, "TRUNCATE TABLE bank_statements CASCADE;")
	return err
}

// PostgresDiscrepancyRepo provides PostgreSQL persistence for detected financial discrepancies.
type PostgresDiscrepancyRepo struct {
	pool *pgxpool.Pool
}

func (r *PostgresDiscrepancyRepo) InsertBatch(ctx context.Context, discrepancies []domain.Discrepancy) error {
	if len(discrepancies) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO discrepancies (
			id, order_id, payout_id, discrepancy_level, discrepancy_type,
			expected_amount, actual_amount, mismatch_amount, severity,
			raw_evidence, ai_diagnosis, status, detected_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO UPDATE SET
			expected_amount = EXCLUDED.expected_amount,
			actual_amount = EXCLUDED.actual_amount,
			mismatch_amount = EXCLUDED.mismatch_amount,
			severity = EXCLUDED.severity,
			raw_evidence = EXCLUDED.raw_evidence,
			ai_diagnosis = EXCLUDED.ai_diagnosis,
			status = EXCLUDED.status;
	`
	for _, d := range discrepancies {
		rawEvidenceJSON, err := json.Marshal(d.RawEvidence)
		if err != nil {
			return fmt.Errorf("marshal raw evidence for discrepancy %s: %w", d.ID, err)
		}

		var aiDiagnosisJSON []byte
		if d.AIDiagnosis != nil {
			aiDiagnosisJSON, err = json.Marshal(d.AIDiagnosis)
			if err != nil {
				return fmt.Errorf("marshal ai diagnosis for discrepancy %s: %w", d.ID, err)
			}
		}

		batch.Queue(
			query,
			d.ID,
			d.OrderID,
			d.PayoutID,
			string(d.DiscrepancyLevel),
			string(d.DiscrepancyType),
			d.ExpectedAmount,
			d.ActualAmount,
			d.MismatchAmount,
			string(d.Severity),
			rawEvidenceJSON,
			aiDiagnosisJSON,
			string(d.Status),
			d.DetectedAt,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range discrepancies {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert discrepancy batch exec: %w", err)
		}
	}
	return nil
}

func (r *PostgresDiscrepancyRepo) FindAll(ctx context.Context, filter DiscrepancyFilter) ([]domain.Discrepancy, error) {
	query := `
		SELECT id, order_id, payout_id, discrepancy_level, discrepancy_type,
		       expected_amount, actual_amount, mismatch_amount, severity,
		       raw_evidence, ai_diagnosis, status, detected_at
		FROM discrepancies
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if filter.Level != nil {
		query += fmt.Sprintf(" AND discrepancy_level = $%d", argIdx)
		args = append(args, string(*filter.Level))
		argIdx++
	}
	if filter.Type != nil {
		query += fmt.Sprintf(" AND discrepancy_type = $%d", argIdx)
		args = append(args, string(*filter.Type))
		argIdx++
	}
	if filter.Severity != nil {
		query += fmt.Sprintf(" AND severity = $%d", argIdx)
		args = append(args, string(*filter.Severity))
		argIdx++
	}
	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, string(*filter.Status))
		argIdx++
	}
	if filter.OrderID != nil {
		query += fmt.Sprintf(" AND order_id = $%d", argIdx)
		args = append(args, *filter.OrderID)
		argIdx++
	}
	if filter.PayoutID != nil {
		query += fmt.Sprintf(" AND payout_id = $%d", argIdx)
		args = append(args, *filter.PayoutID)
		argIdx++
	}

	query += " ORDER BY detected_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
		argIdx++
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query discrepancies: %w", err)
	}
	defer rows.Close()

	var discrepancies []domain.Discrepancy
	for rows.Next() {
		var d domain.Discrepancy
		var levelStr, typeStr, sevStr, statusStr string
		var rawEvidenceJSON, aiDiagnosisJSON []byte

		if err := rows.Scan(
			&d.ID,
			&d.OrderID,
			&d.PayoutID,
			&levelStr,
			&typeStr,
			&d.ExpectedAmount,
			&d.ActualAmount,
			&d.MismatchAmount,
			&sevStr,
			&rawEvidenceJSON,
			&aiDiagnosisJSON,
			&statusStr,
			&d.DetectedAt,
		); err != nil {
			return nil, fmt.Errorf("scan discrepancy: %w", err)
		}

		d.DiscrepancyLevel = domain.DiscrepancyLevel(levelStr)
		d.DiscrepancyType = domain.DiscrepancyType(typeStr)
		d.Severity = domain.DiscrepancySeverity(sevStr)
		d.Status = domain.DiscrepancyStatus(statusStr)

		if len(rawEvidenceJSON) > 0 {
			_ = json.Unmarshal(rawEvidenceJSON, &d.RawEvidence)
		}
		if len(aiDiagnosisJSON) > 0 {
			var diag domain.AIDiagnosis
			if err := json.Unmarshal(aiDiagnosisJSON, &diag); err == nil {
				d.AIDiagnosis = &diag
			}
		}
		discrepancies = append(discrepancies, d)
	}
	return discrepancies, rows.Err()
}

func (r *PostgresDiscrepancyRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Discrepancy, error) {
	query := `
		SELECT id, order_id, payout_id, discrepancy_level, discrepancy_type,
		       expected_amount, actual_amount, mismatch_amount, severity,
		       raw_evidence, ai_diagnosis, status, detected_at
		FROM discrepancies
		WHERE id = $1;
	`
	var d domain.Discrepancy
	var levelStr, typeStr, sevStr, statusStr string
	var rawEvidenceJSON, aiDiagnosisJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&d.ID,
		&d.OrderID,
		&d.PayoutID,
		&levelStr,
		&typeStr,
		&d.ExpectedAmount,
		&d.ActualAmount,
		&d.MismatchAmount,
		&sevStr,
		&rawEvidenceJSON,
		&aiDiagnosisJSON,
		&statusStr,
		&d.DetectedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find discrepancy by id: %w", err)
	}

	d.DiscrepancyLevel = domain.DiscrepancyLevel(levelStr)
	d.DiscrepancyType = domain.DiscrepancyType(typeStr)
	d.Severity = domain.DiscrepancySeverity(sevStr)
	d.Status = domain.DiscrepancyStatus(statusStr)

	if len(rawEvidenceJSON) > 0 {
		_ = json.Unmarshal(rawEvidenceJSON, &d.RawEvidence)
	}
	if len(aiDiagnosisJSON) > 0 {
		var diag domain.AIDiagnosis
		if err := json.Unmarshal(aiDiagnosisJSON, &diag); err == nil {
			d.AIDiagnosis = &diag
		}
	}
	return &d, nil
}

func (r *PostgresDiscrepancyRepo) FindByOrderID(ctx context.Context, orderID string) (*domain.Discrepancy, error) {
	query := `
		SELECT id, order_id, payout_id, discrepancy_level, discrepancy_type,
		       expected_amount, actual_amount, mismatch_amount, severity,
		       raw_evidence, ai_diagnosis, status, detected_at
		FROM discrepancies
		WHERE order_id = $1
		ORDER BY detected_at DESC
		LIMIT 1;
	`
	var d domain.Discrepancy
	var levelStr, typeStr, sevStr, statusStr string
	var rawEvidenceJSON, aiDiagnosisJSON []byte

	err := r.pool.QueryRow(ctx, query, orderID).Scan(
		&d.ID,
		&d.OrderID,
		&d.PayoutID,
		&levelStr,
		&typeStr,
		&d.ExpectedAmount,
		&d.ActualAmount,
		&d.MismatchAmount,
		&sevStr,
		&rawEvidenceJSON,
		&aiDiagnosisJSON,
		&statusStr,
		&d.DetectedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find discrepancy by order_id: %w", err)
	}

	d.DiscrepancyLevel = domain.DiscrepancyLevel(levelStr)
	d.DiscrepancyType = domain.DiscrepancyType(typeStr)
	d.Severity = domain.DiscrepancySeverity(sevStr)
	d.Status = domain.DiscrepancyStatus(statusStr)

	if len(rawEvidenceJSON) > 0 {
		_ = json.Unmarshal(rawEvidenceJSON, &d.RawEvidence)
	}
	if len(aiDiagnosisJSON) > 0 {
		var diag domain.AIDiagnosis
		if err := json.Unmarshal(aiDiagnosisJSON, &diag); err == nil {
			d.AIDiagnosis = &diag
		}
	}
	return &d, nil
}

func (r *PostgresDiscrepancyRepo) FindByPayoutID(ctx context.Context, payoutID string) (*domain.Discrepancy, error) {
	query := `
		SELECT id, order_id, payout_id, discrepancy_level, discrepancy_type,
		       expected_amount, actual_amount, mismatch_amount, severity,
		       raw_evidence, ai_diagnosis, status, detected_at
		FROM discrepancies
		WHERE payout_id = $1
		ORDER BY detected_at DESC
		LIMIT 1;
	`
	var d domain.Discrepancy
	var levelStr, typeStr, sevStr, statusStr string
	var rawEvidenceJSON, aiDiagnosisJSON []byte

	err := r.pool.QueryRow(ctx, query, payoutID).Scan(
		&d.ID,
		&d.OrderID,
		&d.PayoutID,
		&levelStr,
		&typeStr,
		&d.ExpectedAmount,
		&d.ActualAmount,
		&d.MismatchAmount,
		&sevStr,
		&rawEvidenceJSON,
		&aiDiagnosisJSON,
		&statusStr,
		&d.DetectedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find discrepancy by payout_id: %w", err)
	}

	d.DiscrepancyLevel = domain.DiscrepancyLevel(levelStr)
	d.DiscrepancyType = domain.DiscrepancyType(typeStr)
	d.Severity = domain.DiscrepancySeverity(sevStr)
	d.Status = domain.DiscrepancyStatus(statusStr)

	if len(rawEvidenceJSON) > 0 {
		_ = json.Unmarshal(rawEvidenceJSON, &d.RawEvidence)
	}
	if len(aiDiagnosisJSON) > 0 {
		var diag domain.AIDiagnosis
		if err := json.Unmarshal(aiDiagnosisJSON, &diag); err == nil {
			d.AIDiagnosis = &diag
		}
	}
	return &d, nil
}

func (r *PostgresDiscrepancyRepo) UpdateDiagnosis(ctx context.Context, id uuid.UUID, diagnosis domain.AIDiagnosis) error {
	diagJSON, err := json.Marshal(diagnosis)
	if err != nil {
		return fmt.Errorf("marshal diagnosis: %w", err)
	}

	query := `UPDATE discrepancies SET ai_diagnosis = $1 WHERE id = $2;`
	res, err := r.pool.Exec(ctx, query, diagJSON, id)
	if err != nil {
		return fmt.Errorf("update diagnosis: %w", err)
	}
	if res.RowsAffected() == 0 {
		return errors.New("discrepancy not found")
	}
	return nil
}

func (r *PostgresDiscrepancyRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DiscrepancyStatus) error {
	query := `UPDATE discrepancies SET status = $1 WHERE id = $2;`
	res, err := r.pool.Exec(ctx, query, string(status), id)
	if err != nil {
		return fmt.Errorf("update discrepancy status: %w", err)
	}
	if res.RowsAffected() == 0 {
		return errors.New("discrepancy not found")
	}
	return nil
}

func (r *PostgresDiscrepancyRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM discrepancies;").Scan(&count)
	return count, err
}

func (r *PostgresDiscrepancyRepo) Clear(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, "TRUNCATE TABLE discrepancies CASCADE;")
	return err
}

// PostgresUserRepo provides PostgreSQL persistence for platform users.
type PostgresUserRepo struct {
	pool *pgxpool.Pool
}

func (r *PostgresUserRepo) Create(ctx context.Context, u *domain.User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	now := time.Now().UTC()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}
	if u.Role == "" {
		u.Role = "FINANCE_ANALYST"
	}

	query := `
		INSERT INTO users (id, email, password_hash, full_name, company_name, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (email) DO UPDATE SET
			password_hash = EXCLUDED.password_hash,
			full_name = EXCLUDED.full_name,
			company_name = EXCLUDED.company_name,
			role = EXCLUDED.role,
			updated_at = EXCLUDED.updated_at;
	`
	_, err := r.pool.Exec(ctx, query, u.ID, u.Email, u.PasswordHash, u.FullName, u.CompanyName, u.Role, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *PostgresUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, full_name, company_name, role, created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1);
	`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.CompanyName, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return &u, nil
}

func (r *PostgresUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, full_name, company_name, role, created_at, updated_at
		FROM users
		WHERE id = $1;
	`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.CompanyName, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return &u, nil
}

func (r *PostgresUserRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users;").Scan(&count)
	return count, err
}
