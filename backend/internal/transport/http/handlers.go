package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"reconpilot/internal/domain"
	"reconpilot/internal/ingestion"
	"reconpilot/internal/repository"
	"reconpilot/internal/service"
)

// Handler provides REST endpoints for the reconciliation engine.
type Handler struct {
	reconSvc  *service.ReconciliationService
	exportSvc *service.ExportService
	authSvc   *service.AuthService
}

// NewHandler constructs Handler.
func NewHandler(reconSvc *service.ReconciliationService, exportSvc *service.ExportService, authSvc *service.AuthService) *Handler {
	return &Handler{
		reconSvc:  reconSvc,
		exportSvc: exportSvc,
		authSvc:   authSvc,
	}
}

// Register handles operator account creation.
func (h *Handler) Register(c *gin.Context) {
	if h.authSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "authentication service is not initialized"})
		return
	}

	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registration request", "details": err.Error()})
		return
	}

	resp, err := h.authSvc.Register(c.Request.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrUserAlreadyExists) {
			status = http.StatusConflict
		} else if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "characters") {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// Login handles operator authentication and token generation.
func (h *Handler) Login(c *gin.Context) {
	if h.authSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "authentication service is not initialized"})
		return
	}

	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid login request", "details": err.Error()})
		return
	}

	resp, err := h.authSvc.Login(c.Request.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrInvalidCredentials) {
			status = http.StatusUnauthorized
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetCurrentUser returns the profile of the authenticated operator.
func (h *Handler) GetCurrentUser(c *gin.Context) {
	val, exists := c.Get("user_id")
	if !exists {
		// Fallback for API key clients or system access
		c.JSON(http.StatusOK, domain.UserPublic{
			ID:          uuid.Nil,
			Email:       "operator@reconpilot.io",
			FullName:    "ReconPilot Platform Operator",
			CompanyName: "ReconPilot Production",
			Role:        "LEAD_FINANCE_ARCHITECT",
			CreatedAt:   time.Now().UTC(),
		})
		return
	}

	uid, ok := val.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user identifier in context"})
		return
	}

	if h.authSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "authentication service is not initialized"})
		return
	}

	user, err := h.authSvc.GetCurrentUser(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// HealthCheck responds with server status.
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "UP",
		"version":   "1.0.0",
		"service":   "ReconPilot 3-Way Reconciliation Engine",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// GetSummary returns executive reconciliation KPIs.
func (h *Handler) GetSummary(c *gin.Context) {
	summary, err := h.reconSvc.GetSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute summary", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// GetTraces returns 3-way trace items with search, status filtering, and pagination.
func (h *Handler) GetTraces(c *gin.Context) {
	search := c.Query("search")
	statusFilter := c.Query("status") // "reconciled", "discrepancy", or all
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	traces, total, err := h.reconSvc.GetTraces(c.Request.Context(), search, statusFilter, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch traces", "details": err.Error()})
		return
	}
	if traces == nil {
		traces = []domain.ThreeWayTraceItem{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   traces,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetPayoutBatches returns Level-2 payment gateway payout batch aggregates.
func (h *Handler) GetPayoutBatches(c *gin.Context) {
	batches, err := h.reconSvc.GetPayoutBatches(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch payout batches", "details": err.Error()})
		return
	}
	if batches == nil {
		batches = []domain.PayoutBatchGroup{}
	}
	c.JSON(http.StatusOK, gin.H{
		"data":  batches,
		"total": len(batches),
	})
}

// GetDiscrepancies queries detected financial variances with multi-dimensional filtering.
func (h *Handler) GetDiscrepancies(c *gin.Context) {
	filter := repository.DiscrepancyFilter{}

	if levelStr := c.Query("level"); levelStr != "" {
		l := domain.DiscrepancyLevel(levelStr)
		filter.Level = &l
	}
	if typeStr := c.Query("type"); typeStr != "" {
		t := domain.DiscrepancyType(typeStr)
		filter.Type = &t
	}
	if sevStr := c.Query("severity"); sevStr != "" {
		s := domain.DiscrepancySeverity(sevStr)
		filter.Severity = &s
	}
	if statusStr := c.Query("status"); statusStr != "" {
		st := domain.DiscrepancyStatus(statusStr)
		filter.Status = &st
	}
	if orderID := c.Query("order_id"); orderID != "" {
		filter.OrderID = &orderID
	}
	if payoutID := c.Query("payout_id"); payoutID != "" {
		filter.PayoutID = &payoutID
	}

	limitStr := c.DefaultQuery("limit", "100")
	offsetStr := c.DefaultQuery("offset", "0")
	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	filter.Limit = limit
	filter.Offset = offset

	discrepancies, err := h.reconSvc.GetDiscrepancies(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch discrepancies", "details": err.Error()})
		return
	}
	if discrepancies == nil {
		discrepancies = []domain.Discrepancy{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  discrepancies,
		"total": len(discrepancies),
	})
}

// GetDiscrepancyByID retrieves a single discrepancy audit record.
func (h *Handler) GetDiscrepancyByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid discrepancy uuid format"})
		return
	}

	d, err := h.reconSvc.GetDiscrepancyByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query discrepancy", "details": err.Error()})
		return
	}
	if d == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "discrepancy not found"})
		return
	}

	c.JSON(http.StatusOK, d)
}

// DiagnoseDiscrepancy runs Gemini AI analysis on demand for a discrepancy.
func (h *Handler) DiagnoseDiscrepancy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid discrepancy uuid format"})
		return
	}

	diag, err := h.reconSvc.DiagnoseDiscrepancy(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "diagnosis failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, diag)
}

// UpdateDiscrepancyStatus updates status (e.g. OPEN, IN_DISPUTE, RESOLVED, WRITTEN_OFF).
func (h *Handler) UpdateDiscrepancyStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid discrepancy uuid format"})
		return
	}

	var req struct {
		Status domain.DiscrepancyStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status payload", "details": err.Error()})
		return
	}

	if err := h.reconSvc.UpdateDiscrepancyStatus(c.Request.Context(), id, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated", "id": id, "new_status": req.Status})
}

// TriggerReconciliation triggers an on-demand reconciliation run with optional CSV uploads.
func (h *Handler) TriggerReconciliation(c *gin.Context) {
	var customOrders []domain.Order
	var customPG []domain.PGSettlement
	var customBank []domain.BankStatement

	uploader := "System Operator"
	if email, exists := c.Get("user_email"); exists {
		if eStr, ok := email.(string); ok && eStr != "" {
			uploader = eStr
		}
	} else if name, exists := c.Get("user_full_name"); exists {
		if nStr, ok := name.(string); ok && nStr != "" {
			uploader = nStr
		}
	}

	// 1. Check for orders file
	if fileHeader, err := c.FormFile("orders_file"); err == nil {
		file, err := fileHeader.Open()
		if err == nil {
			defer file.Close()
			rawBytes, _ := io.ReadAll(file)
			parsed, err := ingestion.ParseOrdersCSV(bytes.NewReader(rawBytes), false)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse orders_file CSV", "details": err.Error()})
				return
			}
			customOrders = parsed
			_ = h.reconSvc.RecordUpload(c.Request.Context(), &domain.UploadRecord{
				ID:          uuid.New(),
				UploadedAt:  time.Now().UTC(),
				FileType:    "ORDERS",
				FileName:    fileHeader.Filename,
				FileSize:    fileHeader.Size,
				RecordCount: len(parsed),
				UploadedBy:  uploader,
				RawData:     string(rawBytes),
				Status:      "SUCCESS",
				Summary:     fmt.Sprintf("%d Internal Merchant Orders ingested", len(parsed)),
			})
		}
	}

	// 2. Check for PG settlements file
	if fileHeader, err := c.FormFile("pg_file"); err == nil {
		file, err := fileHeader.Open()
		if err == nil {
			defer file.Close()
			rawBytes, _ := io.ReadAll(file)
			parsed, err := ingestion.ParsePGSettlementsCSV(bytes.NewReader(rawBytes), false)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse pg_file CSV", "details": err.Error()})
				return
			}
			customPG = parsed
			_ = h.reconSvc.RecordUpload(c.Request.Context(), &domain.UploadRecord{
				ID:          uuid.New(),
				UploadedAt:  time.Now().UTC(),
				FileType:    "PG_SETTLEMENTS",
				FileName:    fileHeader.Filename,
				FileSize:    fileHeader.Size,
				RecordCount: len(parsed),
				UploadedBy:  uploader,
				RawData:     string(rawBytes),
				Status:      "SUCCESS",
				Summary:     fmt.Sprintf("%d Payment Gateway Settlements ingested", len(parsed)),
			})
		}
	}

	// 3. Check for bank statement file
	if fileHeader, err := c.FormFile("bank_file"); err == nil {
		file, err := fileHeader.Open()
		if err == nil {
			defer file.Close()
			rawBytes, _ := io.ReadAll(file)
			parsed, err := ingestion.ParseBankStatementsCSV(bytes.NewReader(rawBytes), false)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse bank_file CSV", "details": err.Error()})
				return
			}
			customBank = parsed
			_ = h.reconSvc.RecordUpload(c.Request.Context(), &domain.UploadRecord{
				ID:          uuid.New(),
				UploadedAt:  time.Now().UTC(),
				FileType:    "BANK_STATEMENTS",
				FileName:    fileHeader.Filename,
				FileSize:    fileHeader.Size,
				RecordCount: len(parsed),
				UploadedBy:  uploader,
				RawData:     string(rawBytes),
				Status:      "SUCCESS",
				Summary:     fmt.Sprintf("%d Bank Statement UTR Remittances ingested", len(parsed)),
			})
		}
	}

	// Execute reconciliation
	res, err := h.reconSvc.RunReconciliation(c.Request.Context(), customOrders, customPG, customBank)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "reconciliation execution failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Reconciliation executed successfully",
		"summary":       res.Summary,
		"discrepancies": len(res.Discrepancies),
		"traces":        len(res.TraceItems),
	})
}

// StreamTelemetry streams real-time pipeline events and progress over Server-Sent Events (SSE).
// Keeps the connection persistently open with a 15s heartbeat ticker and X-Accel-Buffering: no.
func (h *Handler) StreamTelemetry(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	ch, unsubscribe := h.reconSvc.Hub().Subscribe()
	defer unsubscribe()

	c.Writer.Flush()

	// Initial connection handshake message
	initEvt := domain.PipelineTelemetryEvent{
		Type:               "connected",
		Stage:              "READY",
		ProgressPercentage: 100.0,
		Message:            "ReconPilot persistent SSE stream established",
		Timestamp:          time.Now().UTC(),
	}
	if initBytes, err := json.Marshal(initEvt); err == nil {
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", string(initBytes))
		c.Writer.Flush()
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	clientGone := c.Request.Context().Done()

	for {
		select {
		case <-clientGone:
			return
		case <-ticker.C:
			// 15s keep-alive heartbeat comment (ignored by JSON parsers, prevents proxy timeouts)
			_, _ = fmt.Fprintf(c.Writer, ": keep-alive\n\n")
			c.Writer.Flush()
		case evt, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(evt)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", string(data))
			c.Writer.Flush()
		}
	}
}

// ResetData completely wipes all orders, settlements, bank statements, and discrepancies.
func (h *Handler) ResetData(c *gin.Context) {
	if err := h.reconSvc.ResetData(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "reset failed", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "All financial ledger data wiped successfully"})
}

// SeedData resets the database and seeds realistic demo data (or erases if mode=clear).
func (h *Handler) SeedData(c *gin.Context) {
	if c.Query("mode") == "clear" {
		if err := h.reconSvc.ResetData(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "reset failed", "details": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "All financial ledger data wiped successfully"})
		return
	}

	if err := h.reconSvc.SeedMockData(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seed failed", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Demo financial dataset successfully seeded"})
}

// ExportReport exports reconciliation results as CSV or JSON.
func (h *Handler) ExportReport(c *gin.Context) {
	format := c.DefaultQuery("format", "csv")
	target := c.DefaultQuery("target", "traces") // "traces" or "discrepancies"

	lastResult := h.reconSvc.LastResult()
	if lastResult == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no reconciliation result available for export"})
		return
	}

	if format == "json" {
		data, err := h.exportSvc.ExportJSON(lastResult)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "export json failed", "details": err.Error()})
			return
		}
		c.Header("Content-Disposition", "attachment; filename=reconpilot-audit.json")
		c.Data(http.StatusOK, "application/json", data)
		return
	}

	// CSV Export
	var data []byte
	var err error
	var filename string

	if target == "discrepancies" {
		data, err = h.exportSvc.ExportDiscrepanciesCSV(lastResult.Discrepancies)
		filename = "reconpilot-discrepancies.csv"
	} else {
		data, err = h.exportSvc.ExportTracesCSV(lastResult.TraceItems)
		filename = "reconpilot-3way-traces.csv"
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "export csv failed", "details": err.Error()})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "text/csv", data)
}

// GetUploadHistory returns all historical data uploads and ingestion records.
func (h *Handler) GetUploadHistory(c *gin.Context) {
	records, err := h.reconSvc.GetUploads(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch upload records", "details": err.Error()})
		return
	}
	if records == nil {
		records = []domain.UploadRecord{}
	}
	c.JSON(http.StatusOK, gin.H{
		"data":  records,
		"total": len(records),
	})
}

// GetUploadRecordByID returns a single upload audit record.
func (h *Handler) GetUploadRecordByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid upload record uuid format"})
		return
	}

	record, err := h.reconSvc.GetUploadByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch upload record", "details": err.Error()})
		return
	}
	if record == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "upload record not found"})
		return
	}

	c.JSON(http.StatusOK, record)
}

// DownloadUploadedFile streams the original CSV payload with standard attachment download headers.
func (h *Handler) DownloadUploadedFile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid upload record uuid format"})
		return
	}

	record, err := h.reconSvc.GetUploadByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch upload record", "details": err.Error()})
		return
	}
	if record == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "upload record not found"})
		return
	}

	fileName := record.FileName
	if !strings.HasSuffix(strings.ToLower(fileName), ".csv") {
		fileName += ".csv"
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(record.RawData))
}

// GetContractRules returns all defined contract rules and matching parameter sets.
func (h *Handler) GetContractRules(c *gin.Context) {
	rules, err := h.reconSvc.GetContractRules(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch contract rules", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rules, "total": len(rules)})
}

// CreateContractRule adds a new contract rate configuration.
func (h *Handler) CreateContractRule(c *gin.Context) {
	var rule domain.ContractRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contract rule payload", "details": err.Error()})
		return
	}

	if rule.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "contract rule name is required"})
		return
	}

	if err := h.reconSvc.CreateContractRule(c.Request.Context(), &rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create contract rule", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// UpdateContractRule updates an existing contract rule.
func (h *Handler) UpdateContractRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contract rule id format"})
		return
	}

	var rule domain.ContractRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contract rule payload", "details": err.Error()})
		return
	}

	rule.ID = id
	if err := h.reconSvc.UpdateContractRule(c.Request.Context(), &rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update contract rule", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// ActivateContractRule activates a specific contract rule and updates matching parameters.
func (h *Handler) ActivateContractRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contract rule id format"})
		return
	}

	rule, err := h.reconSvc.ActivateContractRule(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to activate contract rule", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "activated", "rule": rule})
}

// DeleteContractRule deletes an inactive contract rule.
func (h *Handler) DeleteContractRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contract rule id format"})
		return
	}

	if err := h.reconSvc.DeleteContractRule(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted", "id": id})
}

// ReadFullBody reads io.Reader safely
func readFull(r io.Reader) []byte {
	b, _ := io.ReadAll(r)
	return b
}

