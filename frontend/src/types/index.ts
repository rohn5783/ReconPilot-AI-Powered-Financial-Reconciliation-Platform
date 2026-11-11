export type OrderStatus = 'CAPTURED' | 'PENDING' | 'FAILED' | 'REFUNDED';
export type PayoutStatus = 'SETTLED' | 'PROCESSED' | 'PENDING' | 'FAILED';

export type DiscrepancyLevel = 'LEVEL_1_ORDER_PG' | 'LEVEL_2_PG_BANK';

export type DiscrepancyType =
  | 'FEE_OVERCHARGE'
  | 'TAX_CALCULATION_ERROR'
  | 'ORDER_MISSING_IN_PG'
  | 'PG_ORPHAN_TXN'
  | 'CAPTURE_STATUS_MISMATCH'
  | 'GROSS_AMOUNT_MISMATCH'
  | 'UNSETTLED_PAYOUT'
  | 'BANK_MISSING'
  | 'PAYOUT_SHORTFALL'
  | 'PAYOUT_SURPLUS';

export type DiscrepancySeverity = 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW';
export type DiscrepancyStatus = 'OPEN' | 'IN_DISPUTE' | 'RESOLVED' | 'WRITTEN_OFF';

export interface RawEvidence {
  order?: {
    order_id: string;
    customer_id: string;
    gross_amount: string;
    currency: string;
    order_status: OrderStatus;
    created_at: string;
  };
  pg_settlement?: {
    pg_txn_id: string;
    order_id: string;
    gross_amount: string;
    pg_fee: string;
    tax: string;
    net_amount: string;
    payout_id: string;
    payout_status: PayoutStatus;
    settled_at: string;
  };
  bank_statement?: {
    bank_txn_id: string;
    utr_number: string;
    credit_amount: string;
    description: string;
    transaction_date: string;
  };
  batch_settlements?: Array<{
    pg_txn_id: string;
    order_id: string;
    gross_amount: string;
    pg_fee: string;
    tax: string;
    net_amount: string;
    payout_id: string;
    payout_status: PayoutStatus;
    settled_at: string;
  }>;
  expected_formula: string;
  actual_formula: string;
  calculated_values?: Record<string, string>;
}

export interface AIDiagnosis {
  root_cause_summary: string;
  discrepancy_category: string;
  confidence_score: number;
  recommended_actions: string[];
  draft_dispute_ticket: string;
}

export interface Discrepancy {
  id: string;
  order_id?: string;
  payout_id?: string;
  discrepancy_level: DiscrepancyLevel;
  discrepancy_type: DiscrepancyType;
  expected_amount: string;
  actual_amount: string;
  mismatch_amount: string;
  severity: DiscrepancySeverity;
  raw_evidence: RawEvidence;
  ai_diagnosis?: AIDiagnosis;
  status: DiscrepancyStatus;
  detected_at: string;
}

export interface ThreeWayTraceItem {
  order_id: string;
  order_gross?: string;
  order_status?: OrderStatus;
  pg_txn_id?: string;
  pg_gross?: string;
  pg_fee?: string;
  pg_tax?: string;
  pg_net?: string;
  pg_payout_id?: string;
  pg_payout_status?: PayoutStatus;
  bank_txn_id?: string;
  bank_utr?: string;
  bank_credit?: string;
  is_reconciled: boolean;
  discrepancy?: Discrepancy;
}

export interface PayoutBatchGroup {
  payout_id: string;
  settlements_count: number;
  total_gross: string;
  total_fee: string;
  total_tax: string;
  total_expected_net: string;
  bank_credit?: string;
  bank_utr?: string;
  status: 'MATCHED' | 'SHORTFALL' | 'SURPLUS' | 'BANK_MISSING';
  discrepancy?: Discrepancy;
}

export interface ReconciliationSummary {
  total_orders: number;
  total_gross_order_volume: string;
  total_pg_settlements: number;
  total_pg_fees: string;
  total_pg_tax: string;
  expected_net_payout: string;
  total_bank_credits: string;
  net_discrepancy_variance: string;
  level_1_discrepancies_count: number;
  level_2_discrepancies_count: number;
  three_way_match_rate: string;
  reconciled_orders_count: number;
  unreconciled_orders_count: number;
}

export interface PipelineTelemetryEvent {
  type: 'progress' | 'log' | 'kpi_update' | 'discrepancy' | 'completed';
  stage: 'INGESTION' | 'LEVEL_1_MATCH' | 'LEVEL_2_MATCH' | 'SYNTHESIS' | 'AI_DIAGNOSIS' | 'COMPLETED';
  progress_percentage: number;
  message: string;
  payload?: any;
  timestamp: string;
}

