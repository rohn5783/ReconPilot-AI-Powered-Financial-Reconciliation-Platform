package discovery

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"reconpilot/internal/domain"
	"reconpilot/internal/modules"
	"reconpilot/internal/security"
	"reconpilot/pkg/runner"
	"reconpilot/pkg/validator"
)

// SubfinderJSON represents a single JSON record produced by subfinder -json.
type SubfinderJSON struct {
	Host   string `json:"host"`
	Source string `json:"source"`
	Input  string `json:"input"`
}

// DiscoveryModule discovers subdomains of an authorized root domain.
type DiscoveryModule struct {
	binaryPath string
}

// NewDiscoveryModule creates a new DiscoveryModule.
func NewDiscoveryModule(binaryPath string) *DiscoveryModule {
	if binaryPath == "" {
		binaryPath = "subfinder"
	}
	return &DiscoveryModule{binaryPath: binaryPath}
}

// Name returns the pipeline stage identifier.
func (m *DiscoveryModule) Name() domain.StageName {
	return domain.StageDiscovery
}

// ValidateInput verifies the scope and root domain.
func (m *DiscoveryModule) ValidateInput(stageCtx *modules.StageContext) error {
	if stageCtx.Scope == nil {
		return fmt.Errorf("scope is nil")
	}
	return validator.ValidateFQDN(stageCtx.Scope.RootDomain)
}

// Execute runs subdomain enumeration via subfinder or realistic mock simulation.
func (m *DiscoveryModule) Execute(stageCtx *modules.StageContext) (*modules.StageResult, error) {
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

func (m *DiscoveryModule) executeReal(stageCtx *modules.StageContext, scopeGuard *security.ScopeGuard) (*modules.StageResult, error) {
	rootDomain := stageCtx.Scope.RootDomain
	args := []string{"-d", rootDomain, "-silent", "-json"}

	var mu sync.Mutex
	hostMap := make(map[string]bool)

	// Always include the root domain itself
	hostMap[rootDomain] = true

	emitTelemetryLog(stageCtx, fmt.Sprintf("\x1b[36m[DISCOVERY]\x1b[0m Starting passive subdomain enumeration for target \x1b[1m%s\x1b[0m", rootDomain))

	execCfg := runner.ExecutionConfig{
		Binary: m.binaryPath,
		Args:   args,
		OnStdoutLine: func(line string) {
			line = strings.TrimSpace(line)
			if line == "" {
				return
			}

			var rec SubfinderJSON
			var host string

			if err := json.Unmarshal([]byte(line), &rec); err == nil && rec.Host != "" {
				host = strings.ToLower(strings.TrimSpace(rec.Host))
			} else {
				// Fallback if plain text emitted
				host = strings.ToLower(line)
			}

			if err := validator.ValidateFQDN(host); err != nil {
				return
			}

			inScope, reason := scopeGuard.IsInScope(host)
			if !inScope {
				emitTelemetryLog(stageCtx, fmt.Sprintf("\x1b[33m[SCOPE_FILTER]\x1b[0m Discarded %s: %s", host, reason))
				return
			}

			mu.Lock()
			if !hostMap[host] {
				hostMap[host] = true
				mu.Unlock()

				emitTelemetryLog(stageCtx, fmt.Sprintf("\x1b[32m[DISCOVERED]\x1b[0m \x1b[1m%s\x1b[0m (source: %s)", host, rec.Source))
				emitTelemetryEvent(stageCtx, domain.TelemetryEvent{
					Type:      "asset_discovered",
					ScanJobID: stageCtx.ScanJob.ID,
					Stage:     domain.StageDiscovery,
					Timestamp: time.Now(),
					Payload:   map[string]string{"fqdn": host, "source": rec.Source},
				})
			} else {
				mu.Unlock()
			}
		},
		OnStderrLine: func(line string) {
			emitTelemetryLog(stageCtx, fmt.Sprintf("\x1b[90m[subfinder:err]\x1b[0m %s", line))
		},
	}

	res, err := stageCtx.Runner.Run(stageCtx.Context, execCfg)
	if err != nil && res == nil {
		return nil, fmt.Errorf("subfinder execution failed: %w", err)
	}

	var discovered []string
	for h := range hostMap {
		discovered = append(discovered, h)
	}
	sort.Strings(discovered)

	emitTelemetryLog(stageCtx, fmt.Sprintf("\x1b[32m[DISCOVERY_COMPLETE]\x1b[0m Identified %d verified unique subdomains for %s", len(discovered), rootDomain))

	return &modules.StageResult{
		Stage:           domain.StageDiscovery,
		Success:         true,
		OutputCount:     len(discovered),
		DiscoveredHosts: discovered,
		Metrics: map[string]interface{}{
			"hosts_discovered": len(discovered),
			"duration_ms":      res.Duration.Milliseconds(),
		},
	}, nil
}

func (m *DiscoveryModule) executeMock(stageCtx *modules.StageContext, scopeGuard *security.ScopeGuard) (*modules.StageResult, error) {
	rootDomain := stageCtx.Scope.RootDomain

	emitTelemetryLog(stageCtx, fmt.Sprintf("\x1b[36m[DISCOVERY:DRY_RUN]\x1b[0m Simulating high-density reconnaissance for \x1b[1m%s\x1b[0m", rootDomain))

	mockPrefixes := []struct {
		sub    string
		source string
	}{
		{"", "root_anchor"},
		{"api", "crtsh"},
		{"auth", "virustotal"},
		{"admin", "alienvault"},
		{"portal", "certspotter"},
		{"staging", "github"},
		{"cdn", "threatcrowd"},
		{"mail", "securitytrails"},
		{"vpn", "shodan"},
		{"dev-api", "dnsdumpster"},
		{"internal", "archiveis"},
		{"grafana", "crtsh"},
		{"vault", "censys"},
		{"docs", "sublist3r"},
		{"sso", "certspotter"},
	}

	var discovered []string
	for _, p := range mockPrefixes {
		var fqdn string
		if p.sub == "" {
			fqdn = rootDomain
		} else {
			fqdn = fmt.Sprintf("%s.%s", p.sub, rootDomain)
		}

		inScope, reason := scopeGuard.IsInScope(fqdn)
		if !inScope {
			emitTelemetryLog(stageCtx, fmt.Sprintf("\x1b[33m[SCOPE_FILTER]\x1b[0m Discarded %s: %s", fqdn, reason))
			continue
		}

		discovered = append(discovered, fqdn)
		emitTelemetryLog(stageCtx, fmt.Sprintf("\x1b[32m[DISCOVERED]\x1b[0m \x1b[1m%-30s\x1b[0m source: \x1b[34m%s\x1b[0m", fqdn, p.source))
		emitTelemetryEvent(stageCtx, domain.TelemetryEvent{
			Type:      "asset_discovered",
			ScanJobID: stageCtx.ScanJob.ID,
			Stage:     domain.StageDiscovery,
			Timestamp: time.Now(),
			Payload:   map[string]string{"fqdn": fqdn, "source": p.source},
		})
	}

	emitTelemetryLog(stageCtx, fmt.Sprintf("\x1b[32m[DISCOVERY:DONE]\x1b[0m Discovered %d valid in-scope subdomains", len(discovered)))

	return &modules.StageResult{
		Stage:           domain.StageDiscovery,
		Success:         true,
		OutputCount:     len(discovered),
		DiscoveredHosts: discovered,
		Metrics: map[string]interface{}{
			"hosts_discovered": len(discovered),
			"dry_run":          true,
		},
	}, nil
}

func emitTelemetryLog(stageCtx *modules.StageContext, msg string) {
	if stageCtx.Telemetry == nil {
		return
	}
	stageCtx.Telemetry <- domain.TelemetryEvent{
		Type:      "log",
		ScanJobID: stageCtx.ScanJob.ID,
		Stage:     domain.StageDiscovery,
		Timestamp: time.Now(),
		Payload:   msg,
	}
}

func emitTelemetryEvent(stageCtx *modules.StageContext, event domain.TelemetryEvent) {
	if stageCtx.Telemetry == nil {
		return
	}
	stageCtx.Telemetry <- event
}

