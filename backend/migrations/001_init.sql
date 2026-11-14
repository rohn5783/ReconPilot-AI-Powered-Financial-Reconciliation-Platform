-- ==============================================================================
-- ReconPilot: 3-Way Financial Reconciliation Engine Schema
-- PostgreSQL Migration: 001_init.sql
-- CRITICAL: Strict NUMERIC(18, 4) exact decimal arithmetic (NO FLOAT/REAL)
-- ==============================================================================

-- Enable pgcrypto for gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. Internal Merchant Orders
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

-- 2. Payment Gateway Settlements (Razorpay, Stripe, etc.)
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

-- 3. Bank Statement Credits (HDFC, ICICI, Wire transfers)
CREATE TABLE IF NOT EXISTS bank_statements (
    bank_txn_id VARCHAR(128) PRIMARY KEY,
    utr_number VARCHAR(128) UNIQUE NOT NULL,
    credit_amount NUMERIC(18, 4) NOT NULL CHECK (credit_amount >= 0),
    description TEXT NOT NULL,
    transaction_date TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_bank_statements_utr ON bank_statements(utr_number);
CREATE INDEX IF NOT EXISTS idx_bank_statements_txn_date ON bank_statements(transaction_date DESC);

-- 4. Reconciled Discrepancies & AI Audit Dossiers
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

-- 5. User Authentication & Multi-Role Access Control
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

-- 6. Upload History & Data Ingestion Audit Records
CREATE TABLE IF NOT EXISTS upload_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    file_type VARCHAR(64) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    record_count INT NOT NULL DEFAULT 0,
    uploaded_by VARCHAR(255) NOT NULL DEFAULT 'System Operator',
    raw_data TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'SUCCESS',
    summary TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_upload_records_uploaded_at ON upload_records(uploaded_at DESC);
CREATE INDEX IF NOT EXISTS idx_upload_records_file_type ON upload_records(file_type);

-- 7. Contract Rules & Dynamic Matching Configuration
CREATE TABLE IF NOT EXISTS contract_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(128) NOT NULL,
    mdr_rate NUMERIC(18, 4) NOT NULL DEFAULT 2.0000,
    gst_rate NUMERIC(18, 4) NOT NULL DEFAULT 18.0000,
    fee_tolerance NUMERIC(18, 4) NOT NULL DEFAULT 0.0100,
    rounding_precision INT NOT NULL DEFAULT 4,
    rounding_mode VARCHAR(32) NOT NULL DEFAULT 'HALF_EVEN',
    is_active BOOLEAN NOT NULL DEFAULT false,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_contract_rules_active ON contract_rules(is_active);



