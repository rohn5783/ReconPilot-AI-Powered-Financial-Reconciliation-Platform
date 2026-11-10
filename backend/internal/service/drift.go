package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"reconpilot/internal/domain"
)

// DriftEngine computes differential mutations between scan sessions.
type DriftEngine struct{}

// NewDriftEngine creates a new DriftEngine instance.
func NewDriftEngine() *DriftEngine {
	return &DriftEngine{}
}

// ComputeDrift compares current scan assets and findings against a historical baseline.
func (e *DriftEngine) ComputeDrift(
	currentJobID uuid.UUID,
	previousJobID *uuid.UUID,
	currentAssets []*domain.Asset,
	baselineAssets []*domain.Asset,
	currentFindings []*domain.Finding,
	baselineFindings []*domain.Finding,
) ([]*domain.AssetDrift, *domain.DriftAnalysis) {
	var drifts []*domain.AssetDrift
	analysis := &domain.DriftAnalysis{
		HighRiskMutations: []string{},
	}

	now := time.Now()

	// Map baseline assets by FQDN
	baseAssetMap := make(map[string]*domain.Asset)
	for _, ba := range baselineAssets {
		baseAssetMap[strings.ToLower(ba.FQDN)] = ba
	}

	// Map current assets by FQDN
	curAssetMap := make(map[string]*domain.Asset)
	for _, ca := range currentAssets {
		curAssetMap[strings.ToLower(ca.FQDN)] = ca
	}

	// 1. Detect New Hosts & Port/Tech/IP Mutations
	for fqdn, cur := range curAssetMap {
		base, exists := baseAssetMap[fqdn]
		if !exists {
			// Brand new host detected
			analysis.TotalNewHosts++
			sev := domain.SeverityMedium
			if hasHighRiskOpenPorts(cur.OpenPorts) {
				sev = domain.SeverityHigh
				analysis.HighRiskMutations = append(analysis.HighRiskMutations,
					fmt.Sprintf("New asset %s discovered with exposed high-risk ports %v", fqdn, cur.OpenPorts))
			}

			drifts = append(drifts, &domain.AssetDrift{
				ID:                uuid.New(),
				ScanJobID:         currentJobID,
				PreviousScanJobID: previousJobID,
				AssetID:           &cur.ID,
				FQDN:              cur.FQDN,
				ChangeType:        domain.DriftNewHost,
				Details: map[string]interface{}{
					"fqdn":         cur.FQDN,
					"ip_addresses": cur.IPAddresses,
					"open_ports":   cur.OpenPorts,
					"tech_stack":   cur.TechStack,
					"http_status":  cur.HTTPStatus,
				},
				Severity:   sev,
				DetectedAt: now,
			})
			continue
		}

		// Host exists in both: compare Open Ports
		curPortSet := toPortSet(cur.OpenPorts)
		basePortSet := toPortSet(base.OpenPorts)

		for port := range curPortSet {
			if !basePortSet[port] {
				// Newly opened port
				analysis.TotalOpenedPorts++
				sev := domain.SeverityMedium
				if isCriticalPort(port) {
					sev = domain.SeverityCritical
					analysis.HighRiskMutations = append(analysis.HighRiskMutations,
						fmt.Sprintf("CRITICAL: Port %d opened on %s (potential database or admin exposure)", port, fqdn))
				}

				drifts = append(drifts, &domain.AssetDrift{
					ID:                uuid.New(),
					ScanJobID:         currentJobID,
					PreviousScanJobID: previousJobID,
					AssetID:           &cur.ID,
					FQDN:              cur.FQDN,
					ChangeType:        domain.DriftPortOpened,
					Details: map[string]interface{}{
						"port":           port,
						"previous_ports": base.OpenPorts,
						"current_ports":  cur.OpenPorts,
					},
					Severity:   sev,
					DetectedAt: now,
				})
			}
		}

		for port := range basePortSet {
			if !curPortSet[port] {
				// Closed port
				analysis.TotalClosedPorts++
				drifts = append(drifts, &domain.AssetDrift{
					ID:                uuid.New(),
					ScanJobID:         currentJobID,
					PreviousScanJobID: previousJobID,
					AssetID:           &cur.ID,
					FQDN:              cur.FQDN,
					ChangeType:        domain.DriftPortClosed,
					Details: map[string]interface{}{
						"port":           port,
						"previous_ports": base.OpenPorts,
						"current_ports":  cur.OpenPorts,
					},
					Severity:   domain.SeverityLow,
					DetectedAt: now,
				})
			}
		}

		// Compare Tech Stack
		addedTech, removedTech := diffStringSlices(cur.TechStack, base.TechStack)
		if len(addedTech) > 0 || len(removedTech) > 0 {
			drifts = append(drifts, &domain.AssetDrift{
				ID:                uuid.New(),
				ScanJobID:         currentJobID,
				PreviousScanJobID: previousJobID,
				AssetID:           &cur.ID,
				FQDN:              cur.FQDN,
				ChangeType:        domain.DriftTechChanged,
				Details: map[string]interface{}{
					"added_tech":   addedTech,
					"removed_tech": removedTech,
					"old_stack":    base.TechStack,
					"new_stack":    cur.TechStack,
				},
				Severity:   domain.SeverityLow,
				DetectedAt: now,
			})
		}

		// Compare IP Addresses
		addedIPs, removedIPs := diffStringSlices(cur.IPAddresses, base.IPAddresses)
		if len(addedIPs) > 0 || len(removedIPs) > 0 {
			drifts = append(drifts, &domain.AssetDrift{
				ID:                uuid.New(),
				ScanJobID:         currentJobID,
				PreviousScanJobID: previousJobID,
				AssetID:           &cur.ID,
				FQDN:              cur.FQDN,
				ChangeType:        domain.DriftIPChanged,
				Details: map[string]interface{}{
					"added_ips":   addedIPs,
					"removed_ips": removedIPs,
					"old_ips":     base.IPAddresses,
					"new_ips":     cur.IPAddresses,
				},
				Severity:   domain.SeverityLow,
				DetectedAt: now,
			})
		}
	}

	// 2. Detect Removed Hosts
	for fqdn, base := range baseAssetMap {
		if _, exists := curAssetMap[fqdn]; !exists {
			analysis.TotalRemovedHosts++
			drifts = append(drifts, &domain.AssetDrift{
				ID:                uuid.New(),
				ScanJobID:         currentJobID,
				PreviousScanJobID: previousJobID,
				AssetID:           &base.ID,
				FQDN:              base.FQDN,
				ChangeType:        domain.DriftRemovedHost,
				Details: map[string]interface{}{
					"fqdn": base.FQDN,
				},
				Severity:   domain.SeverityLow,
				DetectedAt: now,
			})
		}
	}

	// 3. Detect Finding Drift (New vs Resolved)
	baseFindingKeys := make(map[string]*domain.Finding)
	for _, bf := range baselineFindings {
		key := fmt.Sprintf("%s:%s", bf.FQDN, bf.RuleID)
		baseFindingKeys[key] = bf
	}

	curFindingKeys := make(map[string]*domain.Finding)
	for _, cf := range currentFindings {
		key := fmt.Sprintf("%s:%s", cf.FQDN, cf.RuleID)
		curFindingKeys[key] = cf

		if _, exists := baseFindingKeys[key]; !exists {
			// Newly introduced vulnerability
			drifts = append(drifts, &domain.AssetDrift{
				ID:                uuid.New(),
				ScanJobID:         currentJobID,
				PreviousScanJobID: previousJobID,
				AssetID:           &cf.AssetID,
				FQDN:              cf.FQDN,
				ChangeType:        domain.DriftFindingAdded,
				Details: map[string]interface{}{
					"rule_id":    cf.RuleID,
					"title":      cf.Title,
					"severity":   cf.Severity,
					"cvss_score": cf.CVSSScore,
				},
				Severity:   cf.Severity,
				DetectedAt: now,
			})
		}
	}

	for key, bf := range baseFindingKeys {
		if _, exists := curFindingKeys[key]; !exists {
			// Previously observed finding is now resolved!
			drifts = append(drifts, &domain.AssetDrift{
				ID:                uuid.New(),
				ScanJobID:         currentJobID,
				PreviousScanJobID: previousJobID,
				AssetID:           &bf.AssetID,
				FQDN:              bf.FQDN,
				ChangeType:        domain.DriftFindingResolved,
				Details: map[string]interface{}{
					"rule_id":           bf.RuleID,
					"title":             bf.Title,
					"previous_severity": bf.Severity,
				},
				Severity:   domain.SeverityInfo,
				DetectedAt: now,
			})
		}
	}

	// Construct comprehensive summary prose for dashboard and AI reporting
	analysis.SummaryDescription = fmt.Sprintf(
		"Scan detected %d new host(s), %d removed host(s), %d newly opened port(s), and %d closed port(s). Found %d critical security mutation(s).",
		analysis.TotalNewHosts, analysis.TotalRemovedHosts, analysis.TotalOpenedPorts, analysis.TotalClosedPorts, len(analysis.HighRiskMutations),
	)

	return drifts, analysis
}

func toPortSet(ports []int) map[int]bool {
	set := make(map[int]bool)
	for _, p := range ports {
		set[p] = true
	}
	return set
}

func hasHighRiskOpenPorts(ports []int) bool {
	for _, p := range ports {
		if isCriticalPort(p) {
			return true
		}
	}
	return false
}

func isCriticalPort(p int) bool {
	switch p {
	case 22, 23, 21, 3306, 5432, 6379, 27017, 9200, 11211:
		return true
	default:
		return false
	}
}

func diffStringSlices(current, baseline []string) (added, removed []string) {
	curMap := make(map[string]bool)
	for _, s := range current {
		curMap[s] = true
	}
	baseMap := make(map[string]bool)
	for _, s := range baseline {
		baseMap[s] = true
	}

	for _, s := range current {
		if !baseMap[s] {
			added = append(added, s)
		}
	}
	for _, s := range baseline {
		if !curMap[s] {
			removed = append(removed, s)
		}
	}

	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

