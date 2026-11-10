package auditor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"reconpilot/internal/domain"
	"reconpilot/internal/modules"
	"reconpilot/pkg/runner"
)

// NucleiJSON represents a single JSON record produced by nuclei -jsonl.
type NucleiJSON struct {
	TemplateID string     `json:"template-id"`
	Info       NucleiInfo `json:"info"`
	Matcher    string     `json:"matcher-name"`
	MatchedAt  string     `json:"matched-at"`
	Request    string     `json:"request"`
	Response   string     `json:"response"`
	Extracted  []string   `json:"extracted-results"`
	CurlCmd    string     `json:"curl-command"`
	Host       string     `json:"host"`
	IP         string     `json:"ip"`
}

type NucleiInfo struct {
	Name           string               `json:"name"`
	Severity       string               `json:"severity"`
	Description    string               `json:"description"`
	Reference      []string             `json:"reference"`
	Remediation    string               `json:"remediation"`
	Classification NucleiClassification `json:"classification"`
}

type NucleiClassification struct {
	CVEID     []string `json:"cve-id"`
	CVSSScore float64  `json:"cvss-score"`
}

// AuditorModule runs vulnerability and misconfiguration checks against live targets.
type AuditorModule struct {
	binaryPath string
}

// NewAuditorModule creates a new AuditorModule.
func NewAuditorModule(binaryPath string) *AuditorModule {
	if binaryPath == "" {
		binaryPath = "nuclei"
	}
	return &AuditorModule{binaryPath: binaryPath}
}

// Name returns the stage identifier.
func (m *AuditorModule) Name() domain.StageName {
	return domain.StageAuditor
}

// ValidateInput ensures there are assets available to audit.
func (m *AuditorModule) ValidateInput(stageCtx *modules.StageContext) error {
	if len(stageCtx.LiveAssets) == 0 {
		return fmt.Errorf("auditor requires at least one live asset from previous stages")
	}
	return nil
}

// Execute performs automated vulnerability assessment.
func (m *AuditorModule) Execute(stageCtx *modules.StageContext) (*modules.StageResult, error) {
	if err := m.ValidateInput(stageCtx); err != nil {
		return nil, err
	}

	if stageCtx.DryRun {
		return m.executeMock(stageCtx)
	}

	return m.executeReal(stageCtx)
}

func (m *AuditorModule) executeReal(stageCtx *modules.StageContext) (*modules.StageResult, error) {
	// Build target asset lookup map by FQDN and IP
	assetMap := make(map[string]*domain.Asset)
	var targets []string

	for _, a := range stageCtx.LiveAssets {
		assetMap[strings.ToLower(a.FQDN)] = a
		targets = append(targets, a.FQDN)
		for _, ip := range a.IPAddresses {
			assetMap[ip] = a
		}
	}

	var stdinBuf bytes.Buffer
	for _, t := range targets {
		stdinBuf.WriteString(t + "\n")
	}

	emitLog(stageCtx, fmt.Sprintf("\x1b[36m[AUDITOR]\x1b[0m Executing security audit templates against %d assets...", len(targets)))

	args := []string{
		"-silent",
		"-jsonl",
		"-severity", "info,low,medium,high,critical",
		"-rate-limit", "50",
		"-timeout", "10",
	}

	var mu sync.Mutex
	var findings []*domain.Finding

	execCfg := runner.ExecutionConfig{
		Binary: m.binaryPath,
		Args:   args,
		Stdin:  &stdinBuf,
		OnStdoutLine: func(line string) {
			line = strings.TrimSpace(line)
			if line == "" {
				return
			}

			var rec NucleiJSON
			if err := json.Unmarshal([]byte(line), &rec); err != nil || rec.TemplateID == "" {
				return
			}

			matchedHost := strings.ToLower(rec.Host)
			if matchedHost == "" && rec.MatchedAt != "" {
				matchedHost = extractHostname(rec.MatchedAt)
			}

			matchedAsset, exists := assetMap[matchedHost]
			var assetID uuid.UUID
			if exists {
				assetID = matchedAsset.ID
			}

			sev := parseSeverity(rec.Info.Severity)
			now := time.Now()

			finding := &domain.Finding{
				ID:          uuid.New(),
				ScanJobID:   stageCtx.ScanJob.ID,
				AssetID:     assetID,
				FQDN:        matchedHost,
				Tool:        "nuclei",
				RuleID:      rec.TemplateID,
				Title:       rec.Info.Name,
				Severity:    sev,
				Description: rec.Info.Description,
				CVEIDs:      rec.Info.Classification.CVEID,
				CVSSScore:   rec.Info.Classification.CVSSScore,
				Remediation: rec.Info.Remediation,
				RawEvidence: domain.RawEvidence{
					MatchedAt:        rec.MatchedAt,
					Request:          rec.Request,
					Response:         rec.Response,
					MatcherName:      rec.Matcher,
					ExtractedResults: rec.Extracted,
					CurlCommand:      rec.CurlCmd,
				},
				Status:    domain.FindingActive,
				CreatedAt: now,
				UpdatedAt: now,
			}

			mu.Lock()
			findings = append(findings, finding)
			mu.Unlock()

			color := severityColor(sev)
			emitLog(stageCtx, fmt.Sprintf("  %s[%-8s]\x1b[0m %-28s \x1b[1m%s\x1b[0m (rule: %s)",
				color, strings.ToUpper(string(sev)), matchedHost, finding.Title, finding.RuleID))

			if stageCtx.Telemetry != nil {
				stageCtx.Telemetry <- domain.TelemetryEvent{
					Type:      "finding_detected",
					ScanJobID: stageCtx.ScanJob.ID,
					Stage:     domain.StageAuditor,
					Timestamp: now,
					Payload:   finding,
				}
			}
		},
		OnStderrLine: func(line string) {
			emitLog(stageCtx, fmt.Sprintf("\x1b[90m[nuclei:err]\x1b[0m %s", line))
		},
	}

	res, err := stageCtx.Runner.Run(stageCtx.Context, execCfg)
	if err != nil && res == nil {
		return nil, fmt.Errorf("nuclei execution failed: %w", err)
	}

	emitLog(stageCtx, fmt.Sprintf("\x1b[32m[AUDITOR_COMPLETE]\x1b[0m Security audit identified %d total findings", len(findings)))

	return &modules.StageResult{
		Stage:       domain.StageAuditor,
		Success:     true,
		OutputCount: len(findings),
		Findings:    findings,
		Metrics: map[string]interface{}{
			"findings_count": len(findings),
			"duration_ms":    res.Duration.Milliseconds(),
		},
	}, nil
}

func (m *AuditorModule) executeMock(stageCtx *modules.StageContext) (*modules.StageResult, error) {
	emitLog(stageCtx, fmt.Sprintf("\x1b[36m[AUDITOR:DRY_RUN]\x1b[0m Simulating full vulnerability audit against %d live assets...", len(stageCtx.LiveAssets)))

	// Map asset by prefix for deterministic mock association
	assetByPrefix := make(map[string]*domain.Asset)
	root := stageCtx.Scope.RootDomain

	for _, a := range stageCtx.LiveAssets {
		if a.FQDN == root {
			assetByPrefix["root"] = a
			continue
		}
		prefix := strings.TrimSuffix(strings.TrimSuffix(a.FQDN, root), ".")
		parts := strings.Split(prefix, ".")
		if len(parts) > 0 {
			assetByPrefix[parts[0]] = a
		}
	}

	now := time.Now()
	var findings []*domain.Finding

	// 1. Critical Finding: Exposed Redis on staging
	if asset, ok := assetByPrefix["staging"]; ok {
		findings = append(findings, &domain.Finding{
			ID:          uuid.New(),
			ScanJobID:   stageCtx.ScanJob.ID,
			AssetID:     asset.ID,
			FQDN:        asset.FQDN,
			Tool:        "nuclei",
			RuleID:      "redis-unauthenticated-access",
			Title:       "Exposed Unauthenticated Redis Cache Service",
			Severity:    domain.SeverityCritical,
			Description: "An unauthenticated Redis server is accessible over the public Internet on TCP port 6379, allowing arbitrary key inspection and potential remote code execution via CONFIG SET.",
			CVEIDs:      []string{"CVE-2022-0543"},
			CVSSScore:   9.8,
			Remediation: "Bind Redis strictly to 127.0.0.1 or internal VPC interface, enforce requirepass in redis.conf, and block TCP 6379 on external firewall security groups.",
			RawEvidence: domain.RawEvidence{
				MatchedAt:   fmt.Sprintf("%s:6379", asset.FQDN),
				Request:     "INFO\r\n",
				Response:    "# Server\r\nredis_version:7.0.5\r\nos:Linux 5.15.0-x86_64\r\ntcp_port:6379\r\n",
				MatcherName: "redis-info-matcher",
				CurlCommand: fmt.Sprintf("nc -zv %s 6379", asset.FQDN),
			},
			Status:    domain.FindingActive,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// 2. High Finding: Exposed .git repository on staging
	if asset, ok := assetByPrefix["staging"]; ok {
		findings = append(findings, &domain.Finding{
			ID:          uuid.New(),
			ScanJobID:   stageCtx.ScanJob.ID,
			AssetID:     asset.ID,
			FQDN:        asset.FQDN,
			Tool:        "nuclei",
			RuleID:      "git-config-disclosure",
			Title:       "Exposed Git Repository Configuration",
			Severity:    domain.SeverityHigh,
			Description: "The .git/config configuration file is publicly accessible over HTTP, exposing repository URLs, branches, internal server addresses, and credential tokens.",
			CVEIDs:      []string{},
			CVSSScore:   7.5,
			Remediation: "Configure Nginx/Apache to deny all requests to dot-prefixed paths (location ~ /\\.git { deny all; return 404; }). Remove .git directory from web production roots.",
			RawEvidence: domain.RawEvidence{
				MatchedAt:   fmt.Sprintf("https://%s/.git/config", asset.FQDN),
				Request:     fmt.Sprintf("GET /.git/config HTTP/1.1\r\nHost: %s\r\nUser-Agent: ReconPilot-Auditor\r\n\r\n", asset.FQDN),
				Response:    "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\n[core]\r\n\trepositoryformatversion = 0\r\n\tfilemode = true\r\n[remote \"origin\"]\r\n\turl = https://github.com/internal-corp/staging-app.git\r\n",
				MatcherName: "git-core-matcher",
				CurlCommand: fmt.Sprintf("curl -isk 'https://%s/.git/config'", asset.FQDN),
			},
			Status:    domain.FindingActive,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// 3. High Finding: Spring Boot Actuator Env on dev-api
	if asset, ok := assetByPrefix["dev-api"]; ok {
		findings = append(findings, &domain.Finding{
			ID:          uuid.New(),
			ScanJobID:   stageCtx.ScanJob.ID,
			AssetID:     asset.ID,
			FQDN:        asset.FQDN,
			Tool:        "nuclei",
			RuleID:      "springboot-actuator-env-leak",
			Title:       "Unprotected Spring Boot Actuator Env Endpoint",
			Severity:    domain.SeverityHigh,
			Description: "The /actuator/env diagnostic endpoint is exposed without authentication, leaking runtime configuration, JVM parameters, and environment variables.",
			CVEIDs:      []string{},
			CVSSScore:   7.5,
			Remediation: "Set management.endpoints.web.exposure.exclude=env in application.properties or require mutual TLS / OAuth2 authentication on /actuator/* routes.",
			RawEvidence: domain.RawEvidence{
				MatchedAt:   fmt.Sprintf("https://%s/actuator/env", asset.FQDN),
				Request:     fmt.Sprintf("GET /actuator/env HTTP/1.1\r\nHost: %s\r\n\r\n", asset.FQDN),
				Response:    "HTTP/1.1 200 OK\r\nContent-Type: application/vnd.spring-boot.actuator.v3+json\r\n\r\n{\"activeProfiles\":[\"dev\"],\"propertySources\":[{\"name\":\"systemEnvironment\"}]}",
				MatcherName: "actuator-env-json",
				CurlCommand: fmt.Sprintf("curl -isk 'https://%s/actuator/env'", asset.FQDN),
			},
			Status:    domain.FindingActive,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// 4. Medium Finding: Overly Permissive CORS on api
	if asset, ok := assetByPrefix["api"]; ok {
		findings = append(findings, &domain.Finding{
			ID:          uuid.New(),
			ScanJobID:   stageCtx.ScanJob.ID,
			AssetID:     asset.ID,
			FQDN:        asset.FQDN,
			Tool:        "nuclei",
			RuleID:      "cors-wildcard-origin",
			Title:       "Permissive CORS Access-Control-Allow-Origin Wildcard",
			Severity:    domain.SeverityMedium,
			Description: "The API endpoint reflects arbitrary Origin headers in Access-Control-Allow-Origin with Access-Control-Allow-Credentials: true, permitting cross-site data theft.",
			CVEIDs:      []string{},
			CVSSScore:   6.5,
			Remediation: "Implement an explicit whitelist of trusted frontend origins. Never blindly reflect the Origin request header with Allow-Credentials enabled.",
			RawEvidence: domain.RawEvidence{
				MatchedAt:   fmt.Sprintf("https://%s/v1/user", asset.FQDN),
				Request:     fmt.Sprintf("OPTIONS /v1/user HTTP/1.1\r\nHost: %s\r\nOrigin: https://evil.attacker.com\r\n\r\n", asset.FQDN),
				Response:    "HTTP/1.1 204 No Content\r\nAccess-Control-Allow-Origin: https://evil.attacker.com\r\nAccess-Control-Allow-Credentials: true\r\n",
				MatcherName: "cors-origin-reflection",
				CurlCommand: fmt.Sprintf("curl -isk -X OPTIONS 'https://%s/v1/user' -H 'Origin: https://evil.attacker.com'", asset.FQDN),
			},
			Status:    domain.FindingActive,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// 5. Medium Finding: Insecure TLS Ciphers on vpn
	if asset, ok := assetByPrefix["vpn"]; ok {
		findings = append(findings, &domain.Finding{
			ID:          uuid.New(),
			ScanJobID:   stageCtx.ScanJob.ID,
			AssetID:     asset.ID,
			FQDN:        asset.FQDN,
			Tool:        "nuclei",
			RuleID:      "tls-cbc-weak-ciphers",
			Title:       "Deprecated TLS CBC Ciphers Supported on VPN Gateway",
			Severity:    domain.SeverityMedium,
			Description: "The TLS configuration permits legacy AES-CBC ciphers susceptible to padding oracle attacks (Lucky 13 / BEAST).",
			CVEIDs:      []string{"CVE-2013-0169"},
			CVSSScore:   5.9,
			Remediation: "Disable TLS 1.0 and 1.1. Enforce TLS 1.3 or TLS 1.2 with AEAD ciphers only (e.g. ECDHE-RSA-AES128-GCM-SHA256, ECDHE-ECDSA-AES256-GCM-SHA384).",
			RawEvidence: domain.RawEvidence{
				MatchedAt:   fmt.Sprintf("%s:443", asset.FQDN),
				Request:     "ClientHello with TLS_RSA_WITH_AES_128_CBC_SHA",
				Response:    "ServerHello accepted cipher TLS_RSA_WITH_AES_128_CBC_SHA",
				MatcherName: "cbc-cipher-matcher",
				CurlCommand: fmt.Sprintf("openssl s_client -connect %s:443 -cipher AES128-SHA", asset.FQDN),
			},
			Status:    domain.FindingActive,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// 6. Low Finding: Missing HSTS Header on portal
	if asset, ok := assetByPrefix["portal"]; ok {
		findings = append(findings, &domain.Finding{
			ID:          uuid.New(),
			ScanJobID:   stageCtx.ScanJob.ID,
			AssetID:     asset.ID,
			FQDN:        asset.FQDN,
			Tool:        "nuclei",
			RuleID:      "missing-strict-transport-security",
			Title:       "Missing Strict-Transport-Security (HSTS) Header",
			Severity:    domain.SeverityLow,
			Description: "The web application does not transmit the Strict-Transport-Security header, leaving users vulnerable to SSL stripping downgrade attacks on insecure networks.",
			CVEIDs:      []string{},
			CVSSScore:   3.7,
			Remediation: "Add Strict-Transport-Security: max-age=63072000; includeSubDomains; preload to all production HTTPS responses.",
			RawEvidence: domain.RawEvidence{
				MatchedAt:   fmt.Sprintf("https://%s/", asset.FQDN),
				Request:     fmt.Sprintf("HEAD / HTTP/1.1\r\nHost: %s\r\n\r\n", asset.FQDN),
				Response:    "HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nServer: Vercel\r\n",
				MatcherName: "hsts-header-absent",
				CurlCommand: fmt.Sprintf("curl -Isk 'https://%s/'", asset.FQDN),
			},
			Status:    domain.FindingActive,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// 7. Info Finding: Detailed Server Version Header on auth
	if asset, ok := assetByPrefix["auth"]; ok {
		findings = append(findings, &domain.Finding{
			ID:          uuid.New(),
			ScanJobID:   stageCtx.ScanJob.ID,
			AssetID:     asset.ID,
			FQDN:        asset.FQDN,
			Tool:        "nuclei",
			RuleID:      "server-header-version-disclosure",
			Title:       "Web Server Exact Version Banner Disclosure",
			Severity:    domain.SeverityInfo,
			Description: "The web server response exposes exact version banner 'Server: nginx/1.24.0', assisting attackers in vulnerability targeting.",
			CVEIDs:      []string{},
			CVSSScore:   0.0,
			Remediation: "Set server_tokens off; in nginx.conf or strip Server header via reverse proxy.",
			RawEvidence: domain.RawEvidence{
				MatchedAt:   fmt.Sprintf("https://%s/", asset.FQDN),
				Request:     fmt.Sprintf("HEAD / HTTP/1.1\r\nHost: %s\r\n\r\n", asset.FQDN),
				Response:    "HTTP/1.1 200 OK\r\nServer: nginx/1.24.0\r\n",
				MatcherName: "server-tokens-exposed",
				CurlCommand: fmt.Sprintf("curl -Isk 'https://%s/'", asset.FQDN),
			},
			Status:    domain.FindingActive,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	for _, f := range findings {
		color := severityColor(f.Severity)
		emitLog(stageCtx, fmt.Sprintf("  %s[%-8s]\x1b[0m \x1b[1m%-28s\x1b[0m %s (rule: %s)",
			color, strings.ToUpper(string(f.Severity)), f.FQDN, f.Title, f.RuleID))

		if stageCtx.Telemetry != nil {
			stageCtx.Telemetry <- domain.TelemetryEvent{
				Type:      "finding_detected",
				ScanJobID: stageCtx.ScanJob.ID,
				Stage:     domain.StageAuditor,
				Timestamp: now,
				Payload:   f,
			}
		}
	}

	emitLog(stageCtx, fmt.Sprintf("\x1b[32m[AUDITOR:DONE]\x1b[0m Identified %d actionable security findings", len(findings)))

	return &modules.StageResult{
		Stage:       domain.StageAuditor,
		Success:     true,
		OutputCount: len(findings),
		Findings:    findings,
		Metrics: map[string]interface{}{
			"findings_count": len(findings),
			"dry_run":        true,
		},
	}, nil
}

func parseSeverity(s string) domain.FindingSeverity {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "critical":
		return domain.SeverityCritical
	case "high":
		return domain.SeverityHigh
	case "medium":
		return domain.SeverityMedium
	case "low":
		return domain.SeverityLow
	default:
		return domain.SeverityInfo
	}
}

func severityColor(sev domain.FindingSeverity) string {
	switch sev {
	case domain.SeverityCritical:
		return "\x1b[31;1m" // Crimson bold
	case domain.SeverityHigh:
		return "\x1b[38;5;208m" // Industrial Orange
	case domain.SeverityMedium:
		return "\x1b[33m" // Amber
	case domain.SeverityLow:
		return "\x1b[34m" // Slate blue
	default:
		return "\x1b[90m" // Gray
	}
}

func extractHostname(rawURL string) string {
	u := strings.TrimPrefix(rawURL, "https://")
	u = strings.TrimPrefix(u, "http://")
	parts := strings.Split(u, "/")
	hostPort := parts[0]
	h := strings.Split(hostPort, ":")[0]
	return strings.ToLower(h)
}

func emitLog(stageCtx *modules.StageContext, msg string) {
	if stageCtx.Telemetry == nil {
		return
	}
	stageCtx.Telemetry <- domain.TelemetryEvent{
		Type:      "log",
		ScanJobID: stageCtx.ScanJob.ID,
		Stage:     domain.StageAuditor,
		Timestamp: time.Now(),
		Payload:   msg,
	}
}

