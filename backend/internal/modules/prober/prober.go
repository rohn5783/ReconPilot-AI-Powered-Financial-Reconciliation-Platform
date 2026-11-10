package prober

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"reconpilot/internal/domain"
	"reconpilot/internal/modules"
	"reconpilot/internal/security"
	"reconpilot/pkg/runner"
)

// HttpxJSON represents streaming JSON output from httpx.
type HttpxJSON struct {
	URL         string            `json:"url"`
	Input       string            `json:"input"`
	Host        string            `json:"host"`
	Port        string            `json:"port"`
	Title       string            `json:"title"`
	StatusCode  int               `json:"status_code"`
	Tech        []string          `json:"technologies"`
	A           []string          `json:"a"`
	ASN         string            `json:"asn"`
	ASNOrg      string            `json:"asn_organization"`
	Server      string            `json:"webserver"`
	Headers     map[string]string `json:"header"`
	TLSData     *HttpxTLS         `json:"tls"`
	Failed      bool              `json:"failed"`
	FinalURL    string            `json:"final_url"`
}

type HttpxTLS struct {
	SubjectDN string   `json:"subject_dn"`
	IssuerDN  string   `json:"issuer_dn"`
	SANs      []string `json:"subject_an"`
	Cipher    string   `json:"cipher"`
	Version   string   `json:"version"`
}

// ProberModule probes discovered hosts to determine live HTTP/HTTPS status and fingerprint tech stacks.
type ProberModule struct {
	binaryPath string
}

// NewProberModule creates a new ProberModule.
func NewProberModule(binaryPath string) *ProberModule {
	if binaryPath == "" {
		binaryPath = "httpx"
	}
	return &ProberModule{binaryPath: binaryPath}
}

// Name returns the stage identifier.
func (m *ProberModule) Name() domain.StageName {
	return domain.StageProbing
}

// ValidateInput verifies the input host list.
func (m *ProberModule) ValidateInput(stageCtx *modules.StageContext) error {
	if len(stageCtx.DiscoveredHosts) == 0 {
		return fmt.Errorf("prober requires at least one discovered host from discovery stage")
	}
	return nil
}

// Execute performs live HTTP probing via httpx or realistic mock simulation.
func (m *ProberModule) Execute(stageCtx *modules.StageContext) (*modules.StageResult, error) {
	if err := m.ValidateInput(stageCtx); err != nil {
		return nil, err
	}

	scopeGuard, err := security.NewScopeGuard(stageCtx.Scope)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize scope guard: %w", err)
	}

	if stageCtx.DryRun {
		return m.executeMock(stageCtx, scopeGuard)
	}

	return m.executeReal(stageCtx, scopeGuard)
}

func (m *ProberModule) executeReal(stageCtx *modules.StageContext, scopeGuard *security.ScopeGuard) (*modules.StageResult, error) {
	args := []string{
		"-silent",
		"-json",
		"-title",
		"-tech-detect",
		"-status-code",
		"-asn",
		"-web-server",
		"-tls-grab",
		"-threads", "25",
		"-timeout", "10",
	}

	// Prepare stdin payload with discovered hosts
	var stdinBuf bytes.Buffer
	for _, h := range stageCtx.DiscoveredHosts {
		stdinBuf.WriteString(h + "\n")
	}

	emitLog(stageCtx, fmt.Sprintf("\x1b[36m[PROBER]\x1b[0m Probing %d targets for live HTTP services...", len(stageCtx.DiscoveredHosts)))

	var mu sync.Mutex
	var liveAssets []*domain.Asset
	seenHosts := make(map[string]bool)

	execCfg := runner.ExecutionConfig{
		Binary: m.binaryPath,
		Args:   args,
		Stdin:  &stdinBuf,
		OnStdoutLine: func(line string) {
			line = strings.TrimSpace(line)
			if line == "" {
				return
			}

			var rec HttpxJSON
			if err := json.Unmarshal([]byte(line), &rec); err != nil || rec.Input == "" {
				return
			}

			fqdn := strings.ToLower(strings.TrimSpace(rec.Input))
			if fqdn == "" && rec.Host != "" {
				fqdn = strings.ToLower(strings.TrimSpace(rec.Host))
			}

			// Validate IP addresses against SSRF blocker
			var validIPs []string
			for _, ipStr := range rec.A {
				parsedIP := net.ParseIP(ipStr)
				if parsedIP == nil {
					continue
				}
				if blocked, reason := security.IsBlockedIP(parsedIP); blocked {
					emitLog(stageCtx, fmt.Sprintf("\x1b[31m[SSRF_BLOCK]\x1b[0m Host %s resolved to blocked IP %s: %s", fqdn, ipStr, reason))
					return
				}
				if inScope, _ := scopeGuard.IsIPInScope(parsedIP); !inScope {
					continue
				}
				validIPs = append(validIPs, ipStr)
			}

			// Assemble tech stack
			techList := rec.Tech
			if rec.Server != "" {
				techList = append(techList, rec.Server)
			}

			// TLS Info
			tlsInfo := domain.TLSDetails{}
			if rec.TLSData != nil {
				tlsInfo.Subject = rec.TLSData.SubjectDN
				tlsInfo.Issuer = rec.TLSData.IssuerDN
				tlsInfo.SANs = rec.TLSData.SANs
				tlsInfo.CipherSuite = rec.TLSData.Cipher
				tlsInfo.Version = rec.TLSData.Version
			}

			now := time.Now()
			asset := &domain.Asset{
				ID:              uuid.New(),
				ScanJobID:       stageCtx.ScanJob.ID,
				ScopeID:         stageCtx.Scope.ID,
				FQDN:            fqdn,
				RootDomain:      stageCtx.Scope.RootDomain,
				IPAddresses:     validIPs,
				ASN:             rec.ASN,
				ASNOrg:          rec.ASNOrg,
				HTTPStatus:      rec.StatusCode,
				PageTitle:       rec.Title,
				TechStack:       techList,
				OpenPorts:       []int{80, 443},
				TLSInfo:         tlsInfo,
				ResponseHeaders: rec.Headers,
				FirstSeenAt:     now,
				LastSeenAt:      now,
				CreatedAt:       now,
				UpdatedAt:       now,
			}

			mu.Lock()
			if !seenHosts[fqdn] {
				seenHosts[fqdn] = true
				liveAssets = append(liveAssets, asset)
				mu.Unlock()

				statusColor := "\x1b[32m"
				if rec.StatusCode >= 400 && rec.StatusCode < 500 {
					statusColor = "\x1b[33m"
				} else if rec.StatusCode >= 500 {
					statusColor = "\x1b[31m"
				}

				emitLog(stageCtx, fmt.Sprintf("\x1b[32m[LIVE]\x1b[0m %-28s %s[%d]\x1b[0m title: %q tech: %v",
					fqdn, statusColor, rec.StatusCode, rec.Title, techList))

				if stageCtx.Telemetry != nil {
					stageCtx.Telemetry <- domain.TelemetryEvent{
						Type:      "asset_probed",
						ScanJobID: stageCtx.ScanJob.ID,
						Stage:     domain.StageProbing,
						Timestamp: now,
						Payload:   asset,
					}
				}
			} else {
				mu.Unlock()
			}
		},
		OnStderrLine: func(line string) {
			emitLog(stageCtx, fmt.Sprintf("\x1b[90m[httpx:err]\x1b[0m %s", line))
		},
	}

	res, err := stageCtx.Runner.Run(stageCtx.Context, execCfg)
	if err != nil && res == nil {
		return nil, fmt.Errorf("httpx probing failed: %w", err)
	}

	emitLog(stageCtx, fmt.Sprintf("\x1b[32m[PROBING_COMPLETE]\x1b[0m Successfully probed %d live HTTP endpoints", len(liveAssets)))

	return &modules.StageResult{
		Stage:       domain.StageProbing,
		Success:     true,
		OutputCount: len(liveAssets),
		Assets:      liveAssets,
		Metrics: map[string]interface{}{
			"live_hosts":  len(liveAssets),
			"duration_ms": res.Duration.Milliseconds(),
		},
	}, nil
}

func (m *ProberModule) executeMock(stageCtx *modules.StageContext, scopeGuard *security.ScopeGuard) (*modules.StageResult, error) {
	emitLog(stageCtx, fmt.Sprintf("\x1b[36m[PROBER:DRY_RUN]\x1b[0m Fingerprinting web services for %d hosts...", len(stageCtx.DiscoveredHosts)))

	// Archetypes for realistic tech stack and HTTP status simulation
	archetypes := map[string]struct {
		status    int
		title     string
		tech      []string
		ips       []string
		asn       string
		asnOrg    string
		ports     []int
		headers   map[string]string
		tlsIssuer string
	}{
		"": {
			status: 200, title: "Official Homepage",
			tech: []string{"Cloudflare", "React 19", "Next.js", "Vercel"},
			ips:  []string{"104.21.15.82", "172.67.140.23"},
			asn:  "AS13335", asnOrg: "CLOUDFLARENET", ports: []int{80, 443},
			headers:   map[string]string{"Server": "cloudflare", "Content-Type": "text/html; charset=utf-8"},
			tlsIssuer: "Cloudflare Inc ECC CA-3",
		},
		"api": {
			status: 200, title: "API Gateway Service",
			tech: []string{"Envoy", "Golang", "gRPC-Web", "Cloudflare"},
			ips:  []string{"104.21.15.82", "172.67.140.23"},
			asn:  "AS13335", asnOrg: "CLOUDFLARENET", ports: []int{80, 443, 8080},
			headers:   map[string]string{"Server": "envoy", "Content-Type": "application/json", "X-Gateway-Route": "v2"},
			tlsIssuer: "Cloudflare Inc ECC CA-3",
		},
		"auth": {
			status: 200, title: "Unified Identity & SSO Portal",
			tech: []string{"Ory Kratos", "React", "Nginx/1.24.0", "AWS ALB"},
			ips:  []string{"52.84.12.33"},
			asn:  "AS16509", asnOrg: "AMAZON-02", ports: []int{80, 443},
			headers:   map[string]string{"Server": "nginx/1.24.0", "X-Frame-Options": "SAMEORIGIN"},
			tlsIssuer: "Amazon RSA 2048 M02",
		},
		"admin": {
			status: 403, title: "403 Forbidden - Internal Access Only",
			tech: []string{"Nginx/1.22.1", "OpenSSL"},
			ips:  []string{"34.120.55.91"},
			asn:  "AS15169", asnOrg: "GOOGLE", ports: []int{443, 8443},
			headers:   map[string]string{"Server": "nginx/1.22.1", "Content-Type": "text/html"},
			tlsIssuer: "Google Trust Services LLC",
		},
		"portal": {
			status: 200, title: "Customer Dashboard & Management",
			tech: []string{"Next.js", "React", "Tailwind CSS", "Node.js"},
			ips:  []string{"76.76.21.21"},
			asn:  "AS13335", asnOrg: "CLOUDFLARENET", ports: []int{80, 443},
			headers:   map[string]string{"X-Powered-By": "Next.js"},
			tlsIssuer: "Let's Encrypt Authority X3",
		},
		"staging": {
			status: 401, title: "401 Authorization Required",
			tech: []string{"Traefik v2.10", "Docker", "Vue.js"},
			ips:  []string{"54.210.88.19"},
			asn:  "AS16509", asnOrg: "AMAZON-02", ports: []int{80, 443, 9090},
			headers:   map[string]string{"WWW-Authenticate": "Basic realm=\"Restricted\""},
			tlsIssuer: "Let's Encrypt Authority X3",
		},
		"grafana": {
			status: 200, title: "Grafana - Enterprise Telemetry",
			tech: []string{"Grafana v10.2.3", "Golang", "React"},
			ips:  []string{"142.93.18.204"},
			asn:  "AS14061", asnOrg: "DIGITALOCEAN-ASN", ports: []int{3000, 443},
			headers:   map[string]string{"X-Grafana-Build": "10.2.3"},
			tlsIssuer: "Let's Encrypt Authority X3",
		},
		"vpn": {
			status: 200, title: "GlobalProtect Portal",
			tech: []string{"Palo Alto Networks GlobalProtect", "Apache"},
			ips:  []string{"198.51.100.45"},
			asn:  "AS3356", asnOrg: "LEVEL3", ports: []int{443, 10443},
			headers:   map[string]string{"Server": "PanWeb"},
			tlsIssuer: "DigiCert Global Root CA",
		},
		"vault": {
			status: 200, title: "HashiCorp Vault UI",
			tech: []string{"HashiCorp Vault", "Ember.js", "Golang"},
			ips:  []string{"35.185.12.190"},
			asn:  "AS15169", asnOrg: "GOOGLE", ports: []int{8200, 443},
			headers:   map[string]string{"X-Vault-Request": "true"},
			tlsIssuer: "Google Trust Services LLC",
		},
	}

	var liveAssets []*domain.Asset
	now := time.Now()
	root := stageCtx.Scope.RootDomain

	for _, fqdn := range stageCtx.DiscoveredHosts {
		prefix := ""
		if fqdn != root {
			prefix = strings.TrimSuffix(strings.TrimSuffix(fqdn, root), ".")
			// Take first label
			parts := strings.Split(prefix, ".")
			if len(parts) > 0 {
				prefix = parts[0]
			}
		}

		arch, found := archetypes[prefix]
		if !found {
			arch = struct {
				status    int
				title     string
				tech      []string
				ips       []string
				asn       string
				asnOrg    string
				ports     []int
				headers   map[string]string
				tlsIssuer string
			}{
				status: 200, title: fmt.Sprintf("Service %s", fqdn),
				tech: []string{"Nginx", "HTML5"},
				ips:  []string{"104.21.15.82"},
				asn:  "AS13335", asnOrg: "CLOUDFLARENET", ports: []int{80, 443},
				headers:   map[string]string{"Server": "nginx"},
				tlsIssuer: "Let's Encrypt Authority X3",
			}
		}

		validUntil := now.Add(90 * 24 * time.Hour)
		validFrom := now.Add(-30 * 24 * time.Hour)

		asset := &domain.Asset{
			ID:          uuid.New(),
			ScanJobID:   stageCtx.ScanJob.ID,
			ScopeID:     stageCtx.Scope.ID,
			FQDN:        fqdn,
			RootDomain:  root,
			IPAddresses: arch.ips,
			ASN:         arch.asn,
			ASNOrg:      arch.asnOrg,
			HTTPStatus:  arch.status,
			PageTitle:   arch.title,
			TechStack:   arch.tech,
			OpenPorts:   arch.ports,
			TLSInfo: domain.TLSDetails{
				Subject:     fmt.Sprintf("CN=%s", fqdn),
				Issuer:      arch.tlsIssuer,
				ValidFrom:   &validFrom,
				ValidUntil:  &validUntil,
				SANs:        []string{fqdn, "*." + root},
				CipherSuite: "TLS_AES_256_GCM_SHA384",
				Version:     "TLSv1.3",
			},
			ResponseHeaders: arch.headers,
			FirstSeenAt:     now,
			LastSeenAt:      now,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		liveAssets = append(liveAssets, asset)

		statusColor := "\x1b[32m"
		if arch.status >= 400 && arch.status < 500 {
			statusColor = "\x1b[33m"
		} else if arch.status >= 500 {
			statusColor = "\x1b[31m"
		}

		emitLog(stageCtx, fmt.Sprintf("\x1b[32m[LIVE]\x1b[0m %-28s %s[%d]\x1b[0m title: %q tech: %v",
			fqdn, statusColor, arch.status, arch.title, arch.tech))

		if stageCtx.Telemetry != nil {
			stageCtx.Telemetry <- domain.TelemetryEvent{
				Type:      "asset_probed",
				ScanJobID: stageCtx.ScanJob.ID,
				Stage:     domain.StageProbing,
				Timestamp: now,
				Payload:   asset,
			}
		}
	}

	emitLog(stageCtx, fmt.Sprintf("\x1b[32m[PROBER:DONE]\x1b[0m Profiled %d responsive HTTP assets", len(liveAssets)))

	return &modules.StageResult{
		Stage:       domain.StageProbing,
		Success:     true,
		OutputCount: len(liveAssets),
		Assets:      liveAssets,
		Metrics: map[string]interface{}{
			"live_hosts": len(liveAssets),
			"dry_run":    true,
		},
	}, nil
}

func emitLog(stageCtx *modules.StageContext, msg string) {
	if stageCtx.Telemetry == nil {
		return
	}
	stageCtx.Telemetry <- domain.TelemetryEvent{
		Type:      "log",
		ScanJobID: stageCtx.ScanJob.ID,
		Stage:     domain.StageProbing,
		Timestamp: time.Now(),
		Payload:   msg,
	}
}

