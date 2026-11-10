package portscan

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"reconpilot/internal/domain"
	"reconpilot/internal/modules"
	"reconpilot/pkg/runner"
)

// NmapXML structs for parsing nmap -oX output
type NmapRun struct {
	XMLName xml.Name   `xml:"nmaprun"`
	Hosts   []NmapHost `xml:"host"`
}

type NmapHost struct {
	Addresses []NmapAddress `xml:"address"`
	Hostnames []NmapHostname `xml:"hostnames>hostname"`
	Ports     []NmapPort    `xml:"ports>port"`
}

type NmapAddress struct {
	Addr     string `xml:"addr,attr"`
	AddrType string `xml:"addrtype,attr"`
}

type NmapHostname struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
}

type NmapPort struct {
	PortID   int         `xml:"portid,attr"`
	Protocol string      `xml:"protocol,attr"`
	State    NmapState   `xml:"state"`
	Service  NmapService `xml:"service"`
}

type NmapState struct {
	State string `xml:"state,attr"`
}

type NmapService struct {
	Name    string `xml:"name,attr"`
	Product string `xml:"product,attr"`
	Version string `xml:"version,attr"`
}

// PortScanModule scans and enumerates open TCP ports and exposed network services.
type PortScanModule struct {
	binaryPath string
}

// NewPortScanModule creates a new PortScanModule.
func NewPortScanModule(binaryPath string) *PortScanModule {
	if binaryPath == "" {
		binaryPath = "nmap"
	}
	return &PortScanModule{binaryPath: binaryPath}
}

// Name returns the stage identifier.
func (m *PortScanModule) Name() domain.StageName {
	return domain.StagePortScan
}

// ValidateInput ensures there are live assets to scan.
func (m *PortScanModule) ValidateInput(stageCtx *modules.StageContext) error {
	if len(stageCtx.LiveAssets) == 0 {
		return fmt.Errorf("portscan requires at least one live asset from prober stage")
	}
	return nil
}

// Execute performs network port discovery.
func (m *PortScanModule) Execute(stageCtx *modules.StageContext) (*modules.StageResult, error) {
	if err := m.ValidateInput(stageCtx); err != nil {
		return nil, err
	}

	if stageCtx.DryRun {
		return m.executeMock(stageCtx)
	}

	return m.executeReal(stageCtx)
}

func (m *PortScanModule) executeReal(stageCtx *modules.StageContext) (*modules.StageResult, error) {
	var targets []string
	targetMap := make(map[string]bool)

	for _, a := range stageCtx.LiveAssets {
		if !targetMap[a.FQDN] {
			targetMap[a.FQDN] = true
			targets = append(targets, a.FQDN)
		}
	}

	emitLog(stageCtx, fmt.Sprintf("\x1b[36m[PORTSCAN]\x1b[0m Starting TCP port scanning across %d live targets...", len(targets)))

	args := []string{
		"-sT",
		"-T4",
		"-Pn",
		"--top-ports", "100",
		"-oX", "-",
	}
	args = append(args, targets...)

	var xmlBuf bytes.Buffer
	execCfg := runner.ExecutionConfig{
		Binary: m.binaryPath,
		Args:   args,
		OnStdoutLine: func(line string) {
			xmlBuf.WriteString(line + "\n")
		},
		OnStderrLine: func(line string) {
			emitLog(stageCtx, fmt.Sprintf("\x1b[90m[nmap:err]\x1b[0m %s", line))
		},
	}

	res, err := stageCtx.Runner.Run(stageCtx.Context, execCfg)
	if err != nil && res == nil {
		return nil, fmt.Errorf("nmap execution failed: %w", err)
	}

	// Parse XML output
	var nmapOutput NmapRun
	if err := xml.Unmarshal(xmlBuf.Bytes(), &nmapOutput); err != nil {
		// If direct XML unmarshal fails, parse partial results
		emitLog(stageCtx, fmt.Sprintf("\x1b[33m[PORTSCAN:WARN]\x1b[0m Failed to fully parse Nmap XML: %v", err))
	}

	// Map discovered ports to assets
	portMap := make(map[string][]int)
	for _, h := range nmapOutput.Hosts {
		var hostnames []string
		for _, hn := range h.Hostnames {
			hostnames = append(hostnames, strings.ToLower(hn.Name))
		}
		var ips []string
		for _, addr := range h.Addresses {
			ips = append(ips, addr.Addr)
		}

		var openPorts []int
		for _, p := range h.Ports {
			if strings.EqualFold(p.State.State, "open") {
				openPorts = append(openPorts, p.PortID)
				serviceName := p.Service.Name
				if serviceName == "" {
					serviceName = "unknown"
				}
				emitLog(stageCtx, fmt.Sprintf("\x1b[32m[PORT_OPEN]\x1b[0m %s:%d/tcp (%s)",
					strings.Join(hostnames, ","), p.PortID, serviceName))
			}
		}

		for _, hn := range hostnames {
			portMap[hn] = append(portMap[hn], openPorts...)
		}
		for _, ip := range ips {
			portMap[ip] = append(portMap[ip], openPorts...)
		}
	}

	// Attach open ports to assets
	var totalPortsDiscovered int
	for _, asset := range stageCtx.LiveAssets {
		if ports, exists := portMap[asset.FQDN]; exists && len(ports) > 0 {
			asset.OpenPorts = deduplicateAndSortPorts(ports)
			totalPortsDiscovered += len(asset.OpenPorts)
		}
	}

	emitLog(stageCtx, fmt.Sprintf("\x1b[32m[PORTSCAN_COMPLETE]\x1b[0m Total open ports discovered: %d", totalPortsDiscovered))

	return &modules.StageResult{
		Stage:       domain.StagePortScan,
		Success:     true,
		OutputCount: totalPortsDiscovered,
		Assets:      stageCtx.LiveAssets,
		Metrics: map[string]interface{}{
			"open_ports":  totalPortsDiscovered,
			"duration_ms": res.Duration.Milliseconds(),
		},
	}, nil
}

func (m *PortScanModule) executeMock(stageCtx *modules.StageContext) (*modules.StageResult, error) {
	emitLog(stageCtx, fmt.Sprintf("\x1b[36m[PORTSCAN:DRY_RUN]\x1b[0m Simulating full-spectrum TCP port reconnaissance for %d assets...", len(stageCtx.LiveAssets)))

	portArchetypes := map[string][]int{
		"api":      {80, 443, 8080, 8443},
		"auth":     {80, 443, 8443},
		"admin":    {443, 8443, 22},
		"portal":   {80, 443},
		"staging":  {80, 443, 9090, 6379}, // High-risk Redis exposed on staging!
		"grafana":  {3000, 443},
		"vpn":      {443, 10443},
		"vault":    {8200, 443},
		"internal": {80, 443, 3306, 5432}, // Database ports on internal
	}

	var totalPortsDiscovered int

	for _, asset := range stageCtx.LiveAssets {
		var assignedPorts []int

		for prefix, ports := range portArchetypes {
			if strings.HasPrefix(asset.FQDN, prefix+".") {
				assignedPorts = ports
				break
			}
		}

		if len(assignedPorts) == 0 {
			assignedPorts = []int{80, 443}
		}

		asset.OpenPorts = deduplicateAndSortPorts(assignedPorts)
		totalPortsDiscovered += len(asset.OpenPorts)

		for _, p := range asset.OpenPorts {
			serviceLabel := portToServiceName(p)
			color := "\x1b[32m"
			if isHighRiskPort(p) {
				color = "\x1b[31;1m" // Red highlight for Redis, MySQL, Postgres, SSH
			}
			emitLog(stageCtx, fmt.Sprintf("  %s[PORT_OPEN]\x1b[0m \x1b[1m%-28s\x1b[0m \x1b[34m%5d/tcp\x1b[0m (%s)",
				color, asset.FQDN, p, serviceLabel))
		}
	}

	emitLog(stageCtx, fmt.Sprintf("\x1b[32m[PORTSCAN:DONE]\x1b[0m Enumerated %d active listening ports", totalPortsDiscovered))

	return &modules.StageResult{
		Stage:       domain.StagePortScan,
		Success:     true,
		OutputCount: totalPortsDiscovered,
		Assets:      stageCtx.LiveAssets,
		Metrics: map[string]interface{}{
			"open_ports": totalPortsDiscovered,
			"dry_run":    true,
		},
	}, nil
}

func deduplicateAndSortPorts(ports []int) []int {
	seen := make(map[int]bool)
	var result []int
	for _, p := range ports {
		if !seen[p] && p > 0 && p <= 65535 {
			seen[p] = true
			result = append(result, p)
		}
	}
	sort.Ints(result)
	return result
}

func isHighRiskPort(port int) bool {
	switch port {
	case 22, 23, 21, 3306, 5432, 6379, 27017, 9200, 11211:
		return true
	default:
		return false
	}
}

func portToServiceName(port int) string {
	switch port {
	case 22:
		return "ssh"
	case 80:
		return "http"
	case 443:
		return "https"
	case 3000:
		return "grafana/node"
	case 3306:
		return "mysql"
	case 5432:
		return "postgresql"
	case 6379:
		return "redis-cache"
	case 8080:
		return "http-proxy"
	case 8200:
		return "vault-api"
	case 8443:
		return "https-alt"
	case 9090:
		return "prometheus/cockpit"
	case 9200:
		return "elasticsearch"
	case 10443:
		return "vpn-gateway"
	default:
		return "custom-service"
	}
}

func emitLog(stageCtx *modules.StageContext, msg string) {
	if stageCtx.Telemetry == nil {
		return
	}
	stageCtx.Telemetry <- domain.TelemetryEvent{
		Type:      "log",
		ScanJobID: stageCtx.ScanJob.ID,
		Stage:     domain.StagePortScan,
		Timestamp: time.Now(),
		Payload:   msg,
	}
}

func _(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

