package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genai"

	"reconpilot/internal/config"
	"reconpilot/internal/domain"
)

// GeminiAnalyst synthesizes raw EASM recon telemetry into structured industrial briefings.
type GeminiAnalyst struct {
	apiKey      string
	model       string
	client      *genai.Client
	optimizer   *TokenOptimizer
	fallback    *FallbackGenerator
	maxTokens   int
	temperature float64
}

// NewGeminiAnalyst initializes the Google GenAI SDK client or fallback engine.
func NewGeminiAnalyst(cfg *config.AIConfig) *GeminiAnalyst {
	analyst := &GeminiAnalyst{
		apiKey:      cfg.GeminiAPIKey,
		model:       cfg.GeminiModel,
		optimizer:   NewTokenOptimizer(),
		fallback:    NewFallbackGenerator(),
		maxTokens:   cfg.MaxTokens,
		temperature: cfg.Temperature,
	}

	if analyst.model == "" {
		analyst.model = "gemini-2.5-flash"
	}

	if cfg.GeminiAPIKey != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  cfg.GeminiAPIKey,
			Backend: genai.BackendGeminiAPI,
		})
		if err == nil {
			analyst.client = client
		}
	}

	return analyst
}

// GenerateReport synthesizes executive summaries, risk scores, and remediation playbooks.
func (a *GeminiAnalyst) GenerateReport(
	ctx context.Context,
	scanJobID uuid.UUID,
	targetDomain string,
	assets []*domain.Asset,
	findings []*domain.Finding,
	drift *domain.DriftAnalysis,
) (*domain.AIReport, error) {
	compactPayload := a.optimizer.Optimize(targetDomain, assets, findings, drift)

	// If no Gemini API key or client not initialized, use deterministic rule-based fallback
	if a.client == nil || a.apiKey == "" {
		return a.fallback.Generate(scanJobID, compactPayload), nil
	}

	// Prepare compact JSON prompt
	payloadBytes, _ := json.Marshal(compactPayload)

	prompt := fmt.Sprintf(`You are a Principal Application Security Architect and Cloud Infrastructure Hardening Specialist.
Analyze the following External Attack Surface Management (EASM) scan telemetry for target domain "%s" and produce a strictly structured JSON executive briefing and remediation playbook.

INPUT SCAN TELEMETRY:
%s

INSTRUCTIONS:
1. "overall_risk_score": integer 0-100 reflecting external posture.
2. "executive_summary": 2-3 concise paragraphs evaluating posture, critical exposures, exposed administrative/database services, and immediate recommendations. Strictly avoid corporate fluff.
3. "attack_surface_drift_analysis": summarize new vs removed hosts and open ports from the input drift.
4. "prioritized_findings": array of findings ranked from Rank 1 upwards, including:
   - "rank": integer
   - "title": string
   - "severity": "critical"|"high"|"medium"|"low"|"info"
   - "affected_asset": string
   - "exploit_probability": string (e.g. "Confirmed Remote / High", "Active Exploit Available", "Opportunistic / Medium")
   - "impact_analysis": string
   - "root_cause": string
5. "remediation_playbook": array of actionable defensive hardening playbooks, including:
   - "title": string
   - "target_category": string (e.g. "Nginx", "Cloudflare", "AWS_SG", "Docker", "TLS", "Firewall")
   - "severity": string
   - "config_snippet": production-ready config file snippet (e.g. nginx.conf block, aws cli command, redis.conf)
   - "explanation": why this defensive control fixes the exposure
   - "validation_steps": array of verification shell commands (e.g. curl, nmap, openssl)

Respond ONLY with a valid JSON object matching the required schema. Do not enclose in markdown code blocks.`, targetDomain, string(payloadBytes))

	// Execute with exponential backoff and jitter
	var lastErr error
	backoff := 1 * time.Second

	for attempt := 0; attempt < 3; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		genConfig := &genai.GenerateContentConfig{
			ResponseMIMEType: "application/json",
		}
		if a.temperature > 0 {
			temp := float32(a.temperature)
			genConfig.Temperature = &temp
		}

		res, err := a.client.Models.GenerateContent(callCtx, a.model, genai.Text(prompt), genConfig)
		cancel()

		if err == nil && res != nil {
			responseText := strings.TrimSpace(res.Text())
			// Strip possible markdown fences if returned
			responseText = strings.TrimPrefix(responseText, "```json")
			responseText = strings.TrimPrefix(responseText, "```")
			responseText = strings.TrimSuffix(responseText, "```")
			responseText = strings.TrimSpace(responseText)

			var report domain.AIReport
			if unmarshalErr := json.Unmarshal([]byte(responseText), &report); unmarshalErr == nil {
				report.ID = uuid.New()
				report.ScanJobID = scanJobID
				report.ModelVersion = a.model
				report.IsFallback = false
				report.GeneratedAt = time.Now()
				if res.UsageMetadata != nil {
					report.TokensUsed = int(res.UsageMetadata.TotalTokenCount)
				}
				return &report, nil
			} else {
				lastErr = fmt.Errorf("failed to unmarshal Gemini JSON output: %w", unmarshalErr)
			}
		} else {
			lastErr = err
		}

		// Wait backoff with jitter before retry
		jitter := time.Duration(rand.Int63n(int64(500 * time.Millisecond)))
		select {
		case <-time.After(backoff + jitter):
			backoff *= 2
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// If all retries exhausted, return graceful fallback report with error logged
	if errors.Is(lastErr, context.Canceled) {
		return nil, lastErr
	}

	fallbackReport := a.fallback.Generate(scanJobID, compactPayload)
	fallbackReport.ExecutiveSummary = fmt.Sprintf("[Note: AI synthesis fell back to deterministic engine due to: %v]\n\n%s",
		lastErr, fallbackReport.ExecutiveSummary)

	return fallbackReport, nil
}

