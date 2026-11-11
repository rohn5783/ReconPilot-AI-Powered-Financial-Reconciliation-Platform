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
	}
	return repo, nil
}

func (r *PostgresRepository) Orders() OrderRepository             { return r.orders }
func (r *PostgresRepository) PG() PGRepository                     { return r.pg }
func (r *PostgresRepository) Bank() BankRepository                 { return r.bank }
func (r *PostgresRepository) Discrepancies() DiscrepancyRepository { return r.discrepancies }

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
