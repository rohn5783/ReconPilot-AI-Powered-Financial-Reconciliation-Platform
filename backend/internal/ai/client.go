package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/genai"

	"reconpilot/internal/domain"
)

// Diagnostician diagnoses financial discrepancies and synthesizes merchant dispute tickets.
type Diagnostician interface {
	Diagnose(ctx context.Context, discrepancy *domain.Discrepancy) (*domain.AIDiagnosis, error)
	DiagnoseBatch(ctx context.Context, discrepancies []domain.Discrepancy) ([]domain.Discrepancy, error)
}

// Client implements Diagnostician using Google GenAI SDK (gemini-2.5-flash) with deterministic fallback.
type Client struct {
	apiKey      string
	model       string
	genaiClient *genai.Client
	mu          sync.RWMutex
}

// NewClient initializes a Google Gemini 2.5 Flash AI Diagnostician.
func NewClient(ctx context.Context, apiKey, model string) (*Client, error) {
	if model == "" {
		model = "gemini-2.5-flash"
	}

	c := &Client{
		apiKey: apiKey,
		model:  model,
	}

	if strings.TrimSpace(apiKey) != "" && apiKey != "<KEY_HERE>" {
		cfg := &genai.ClientConfig{
			APIKey: apiKey,
		}
		gClient, err := genai.NewClient(ctx, cfg)
		if err == nil {
			c.genaiClient = gClient
		}
	}

	return c, nil
}

// Diagnose synthesizes an AI root cause analysis and dispute ticket for a discrepancy.
func (c *Client) Diagnose(ctx context.Context, d *domain.Discrepancy) (*domain.AIDiagnosis, error) {
	if d == nil {
		return nil, fmt.Errorf("discrepancy cannot be nil")
	}

	// If no Gemini client is available or in offline mode, use deterministic rule-based engine
	if c.genaiClient == nil || c.apiKey == "" || c.apiKey == "<KEY_HERE>" {
		return RuleBasedDiagnose(d), nil
	}

	prompt := c.buildPrompt(d)

	// Call Gemini with a 15-second timeout
	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	resp, err := c.genaiClient.Models.GenerateContent(callCtx, c.model, genai.Text(prompt), &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
	})
	if err != nil {
		// Log and gracefully fall back to deterministic engine
		return RuleBasedDiagnose(d), nil
	}

	responseText := resp.Text()
	if responseText == "" {
		return RuleBasedDiagnose(d), nil
	}

	// Clean Markdown code fence if model wrapped JSON in ```json ... ```
	cleaned := strings.TrimSpace(responseText)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var diag domain.AIDiagnosis
	if err := json.Unmarshal([]byte(cleaned), &diag); err != nil {
		return RuleBasedDiagnose(d), nil
	}

	if diag.ConfidenceScore <= 0 || diag.ConfidenceScore > 100 {
		diag.ConfidenceScore = 92
	}

	return &diag, nil
}

// DiagnoseBatch analyzes a slice of discrepancies concurrently.
func (c *Client) DiagnoseBatch(ctx context.Context, discrepancies []domain.Discrepancy) ([]domain.Discrepancy, error) {
	res := make([]domain.Discrepancy, len(discrepancies))
	copy(res, discrepancies)

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // max 5 concurrent Gemini calls

	for i := range res {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			diag, err := c.Diagnose(ctx, &res[idx])
			if err == nil && diag != nil {
				res[idx].AIDiagnosis = diag
			}
		}(i)
	}

	wg.Wait()
	return res, nil
}

func (c *Client) buildPrompt(d *domain.Discrepancy) string {
	evidenceJSON, _ := json.MarshalIndent(d.RawEvidence, "", "  ")

	return fmt.Sprintf(`You are a Senior Fintech Forensic Accountant and Payment Operations Auditor.
Analyze the following financial reconciliation discrepancy detected between an internal order ledger, a payment gateway settlement, and a bank nodal account.

DISCREPANCY SPECIFICATION:
- Discrepancy Level: %s
- Discrepancy Classification: %s
- Severity: %s
- Expected Amount: ₹%s
- Actual Amount: ₹%s
- Mismatch Variance: ₹%s

EVIDENCE DOSSIER (FORMULAS & LEDGER RECORDS):
%s

TASK:
Produce an institutional root cause analysis and a formal dispute ticket for the payment gateway or bank operations team.

STRICT JSON OUTPUT FORMAT (Return ONLY this JSON, no preamble):
{
  "root_cause_summary": "Precise 1-2 sentence technical explanation of why this financial divergence occurred.",
  "discrepancy_category": "Short 2-5 word financial category (e.g. Gateway MDR Overcharge, Uncredited Remittance)",
  "confidence_score": 95,
  "recommended_actions": [
    "Action step 1",
    "Action step 2"
  ],
  "draft_dispute_ticket": "Full professional dispute ticket ready to copy-paste to Razorpay/Stripe/Bank helpdesk including references, exact formulas, numbers, and demanded resolution."
}`,
		d.DiscrepancyLevel,
		d.DiscrepancyType,
		d.Severity,
		d.ExpectedAmount.StringFixed(4),
		d.ActualAmount.StringFixed(4),
		d.MismatchAmount.StringFixed(4),
		string(evidenceJSON),
	)
}

