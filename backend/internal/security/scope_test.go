package security

import (
	"net"
	"testing"

	"reconpilot/internal/domain"
)

func TestScopeGuardSubdomains(t *testing.T) {
	scope := &domain.Scope{
		RootDomain: "example.com",
		ExcludedSubdomains: []string{
			"mail.example.com",
			"*.internal.example.com",
		},
	}

	guard, err := NewScopeGuard(scope)
	if err != nil {
		t.Fatalf("failed to create scope guard: %v", err)
	}

	// In-scope targets
	allowedTargets := []string{
		"example.com",
		"api.example.com",
		"dev.stage.example.com",
		"static.assets.example.com",
	}

	for _, target := range allowedTargets {
		inScope, reason := guard.IsInScope(target)
		if !inScope {
			t.Errorf("expected %q to be IN scope, but got: %s", target, reason)
		}
	}

	// Out-of-scope targets
	blockedTargets := []struct {
		target string
		desc   string
	}{
		{"attacker.com", "completely different domain"},
		{"example.com.attacker.com", "attacker subdomain spoofing root"},
		{"notexample.com", "partial root match"},
		{"mail.example.com", "explicitly excluded subdomain"},
		{"db.internal.example.com", "wildcard excluded subdomain (*.internal.example.com)"},
		{"redis.db.internal.example.com", "nested wildcard excluded subdomain"},
	}

	for _, tc := range blockedTargets {
		inScope, _ := guard.IsInScope(tc.target)
		if inScope {
			t.Errorf("expected %q (%s) to be OUT OF SCOPE, but was allowed", tc.target, tc.desc)
		}
	}
}

func TestScopeGuardIPs(t *testing.T) {
	scope := &domain.Scope{
		RootDomain: "example.com",
		AllowedCIDRs: []string{
			"203.0.113.0/24",
			"198.51.100.0/24",
		},
		ExcludedCIDRs: []string{
			"203.0.113.50/32",
		},
	}

	guard, err := NewScopeGuard(scope)
	if err != nil {
		t.Fatalf("failed to create scope guard: %v", err)
	}

	// Allowed IP
	ip1 := net.ParseIP("203.0.113.10")
	if inScope, reason := guard.IsIPInScope(ip1); !inScope {
		t.Errorf("expected %s to be in scope, got: %s", ip1, reason)
	}

	// Excluded IP within allowed CIDR
	ip2 := net.ParseIP("203.0.113.50")
	if inScope, _ := guard.IsIPInScope(ip2); inScope {
		t.Errorf("expected %s to be excluded by ExcludedCIDRs", ip2)
	}

	// Outside allowed CIDRs
	ip3 := net.ParseIP("8.8.8.8")
	if inScope, _ := guard.IsIPInScope(ip3); inScope {
		t.Errorf("expected %s to be rejected as outside allowed CIDRs", ip3)
	}
}

