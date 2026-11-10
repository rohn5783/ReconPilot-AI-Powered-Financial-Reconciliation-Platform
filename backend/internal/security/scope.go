package security

import (
	"fmt"
	"net"
	"strings"

	"reconpilot/internal/domain"
	"reconpilot/pkg/validator"
)

// ScopeGuard enforces target authorization rules against configured scope policies.
type ScopeGuard struct {
	scope *domain.Scope
}

// NewScopeGuard creates a ScopeGuard for a given scope.
func NewScopeGuard(scope *domain.Scope) (*ScopeGuard, error) {
	if scope == nil {
		return nil, fmt.Errorf("scope cannot be nil")
	}

	if err := validator.ValidateFQDN(scope.RootDomain); err != nil {
		return nil, fmt.Errorf("invalid root domain in scope: %w", err)
	}

	return &ScopeGuard{scope: scope}, nil
}

// IsInScope checks whether candidateFQDN is within the authorized scope boundaries.
func (g *ScopeGuard) IsInScope(candidateFQDN string) (bool, string) {
	fqdn := strings.ToLower(strings.TrimSpace(candidateFQDN))
	fqdn = strings.TrimSuffix(fqdn, ".")
	root := strings.ToLower(strings.TrimSpace(g.scope.RootDomain))
	root = strings.TrimSuffix(root, ".")

	if fqdn == "" {
		return false, "empty domain name"
	}

	// Must either be exact root domain or a subdomain of root domain
	if fqdn != root && !strings.HasSuffix(fqdn, "."+root) {
		return false, fmt.Sprintf("target %q does not belong to authorized root domain %q", fqdn, root)
	}

	// Check excluded subdomains
	for _, excluded := range g.scope.ExcludedSubdomains {
		ex := strings.ToLower(strings.TrimSpace(excluded))
		ex = strings.TrimSuffix(ex, ".")

		// Wildcard check (*.internal.example.com)
		if strings.HasPrefix(ex, "*.") {
			suffix := strings.TrimPrefix(ex, "*.")
			if fqdn == suffix || strings.HasSuffix(fqdn, "."+suffix) {
				return false, fmt.Sprintf("target %q matches wildcard excluded subdomain %q", fqdn, excluded)
			}
		} else if fqdn == ex {
			return false, fmt.Sprintf("target %q is explicitly excluded", fqdn)
		}
	}

	return true, ""
}

// IsIPInScope validates that discovered or resolved IPs respect CIDR boundaries.
func (g *ScopeGuard) IsIPInScope(ip net.IP) (bool, string) {
	if ip == nil {
		return false, "nil IP"
	}

	// 1. Check excluded CIDRs
	for _, exCIDR := range g.scope.ExcludedCIDRs {
		netCIDR, err := validator.ValidateCIDR(exCIDR)
		if err == nil && netCIDR.Contains(ip) {
			return false, fmt.Sprintf("IP %s falls into excluded CIDR %s", ip.String(), exCIDR)
		}
	}

	// 2. If allowed CIDRs is defined and non-empty, IP MUST match at least one
	if len(g.scope.AllowedCIDRs) > 0 {
		matched := false
		for _, allowedCIDR := range g.scope.AllowedCIDRs {
			netCIDR, err := validator.ValidateCIDR(allowedCIDR)
			if err == nil && netCIDR.Contains(ip) {
				matched = true
				break
			}
		}
		if !matched {
			return false, fmt.Sprintf("IP %s does not match any allowed scope CIDR blocks", ip.String())
		}
	}

	return true, ""
}

