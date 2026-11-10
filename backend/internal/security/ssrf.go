package security

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// BlockedNetwork defines a blocked CIDR block with an explanatory reason.
type BlockedNetwork struct {
	Net    *net.IPNet
	Reason string
}

var blockedIPv4CIDRs = []struct {
	cidr   string
	reason string
}{
	// IPv4 Private & Reserved (RFC 1122, RFC 1918, RFC 3927, RFC 5735, RFC 6598)
	{"0.0.0.0/8", "Current network (RFC 1122)"},
	{"10.0.0.0/8", "Private-Use network (RFC 1918)"},
	{"100.64.0.0/10", "Shared Address Space / CGNAT (RFC 6598)"},
	{"127.0.0.0/8", "Loopback address range (RFC 1122)"},
	{"169.254.0.0/16", "Link-Local / Cloud Metadata range (RFC 3927 / AWS / GCP)"},
	{"172.16.0.0/12", "Private-Use network (RFC 1918)"},
	{"192.0.0.0/24", "IETF Protocol Assignments (RFC 6890)"},
	{"192.0.2.0/24", "Documentation / TEST-NET-1 (RFC 5737)"},
	{"192.88.99.0/24", "6to4 Relay Anycast (RFC 7526)"},
	{"192.168.0.0/16", "Private-Use network (RFC 1918)"},
	{"198.18.0.0/15", "Network benchmark tests (RFC 2544)"},
	{"198.51.100.0/24", "Documentation / TEST-NET-2 (RFC 5737)"},
	{"203.0.113.0/24", "Documentation / TEST-NET-3 (RFC 5737)"},
	{"224.0.0.0/4", "Multicast (RFC 5771)"},
	{"240.0.0.0/4", "Reserved for Future Use (RFC 1112)"},
	{"255.255.255.255/32", "Limited Broadcast"},
}

var blockedIPv6CIDRs = []struct {
	cidr   string
	reason string
}{
	// IPv6 Private & Special Use (RFC 4291, RFC 4193, RFC 6666, RFC 3849)
	{"::/128", "Unspecified address"},
	{"::1/128", "IPv6 Loopback"},
	{"100::/64", "Discard-Only prefix (RFC 6666)"},
	{"2001:db8::/32", "Documentation prefix (RFC 3849)"},
	{"fc00::/7", "Unique Local Address / ULA (RFC 4193)"},
	{"fe80::/10", "Link-Local Unicast (RFC 4291)"},
	{"ff00::/8", "IPv6 Multicast (RFC 4291)"},
}

var (
	compiledBlockedIPv4 []BlockedNetwork
	compiledBlockedIPv6 []BlockedNetwork
)

func init() {
	for _, entry := range blockedIPv4CIDRs {
		_, ipNet, err := net.ParseCIDR(entry.cidr)
		if err != nil {
			panic(fmt.Sprintf("internal security init: invalid IPv4 CIDR %q: %v", entry.cidr, err))
		}
		compiledBlockedIPv4 = append(compiledBlockedIPv4, BlockedNetwork{
			Net:    ipNet,
			Reason: entry.reason,
		})
	}

	for _, entry := range blockedIPv6CIDRs {
		_, ipNet, err := net.ParseCIDR(entry.cidr)
		if err != nil {
			panic(fmt.Sprintf("internal security init: invalid IPv6 CIDR %q: %v", entry.cidr, err))
		}
		compiledBlockedIPv6 = append(compiledBlockedIPv6, BlockedNetwork{
			Net:    ipNet,
			Reason: entry.reason,
		})
	}
}

// IsBlockedIP inspects an IP address and determines if it belongs to private, loopback,
// link-local, cloud-metadata, or reserved ranges.
func IsBlockedIP(ip net.IP) (bool, string) {
	if ip == nil {
		return true, "nil IP address"
	}

	// Unwrap IPv4-mapped IPv6 address (e.g. ::ffff:127.0.0.1 or ::ffff:10.0.0.1)
	if v4 := ip.To4(); v4 != nil {
		// Built-in standard library checks for IPv4
		if v4.IsLoopback() {
			return true, "Loopback address (SSRF risk)"
		}
		if v4.IsPrivate() {
			return true, "RFC 1918 Private network address (SSRF risk)"
		}
		if v4.IsLinkLocalUnicast() || v4.IsLinkLocalMulticast() {
			return true, "Link-local address (Metadata SSRF risk)"
		}
		if v4.IsMulticast() {
			return true, "Multicast address"
		}
		if v4.IsUnspecified() {
			return true, "Unspecified 0.0.0.0 address"
		}

		// Check against compiled IPv4 subnet table
		for _, b := range compiledBlockedIPv4 {
			if b.Net.Contains(v4) {
				return true, fmt.Sprintf("%s (%s)", b.Reason, b.Net.String())
			}
		}

		return false, ""
	}

	// Native IPv6 checks
	if ip.IsLoopback() {
		return true, "IPv6 Loopback address (SSRF risk)"
	}
	if ip.IsPrivate() {
		return true, "IPv6 Private / Unique Local address (SSRF risk)"
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true, "IPv6 Link-local address (SSRF risk)"
	}
	if ip.IsMulticast() {
		return true, "IPv6 Multicast address"
	}
	if ip.IsUnspecified() {
		return true, "IPv6 Unspecified :: address"
	}

	// Check against compiled IPv6 subnet table
	for _, b := range compiledBlockedIPv6 {
		if b.Net.Contains(ip) {
			return true, fmt.Sprintf("%s (%s)", b.Reason, b.Net.String())
		}
	}

	return false, ""
}

// HostValidationResult holds DNS resolution and SSRF verification output.
type HostValidationResult struct {
	Host         string   `json:"host"`
	ResolvedIPs  []net.IP `json:"resolved_ips"`
	IPStrings    []string `json:"ip_strings"`
	IsPublicOnly bool     `json:"is_public_only"`
}

// SSRFValidator provides target verification and resolution guardrails.
type SSRFValidator struct {
	resolver *net.Resolver
	timeout  time.Duration
}

// NewSSRFValidator creates a validator with explicit DNS timeout.
func NewSSRFValidator(timeout time.Duration) *SSRFValidator {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &SSRFValidator{
		resolver: net.DefaultResolver,
		timeout:  timeout,
	}
}

// ValidateTarget verifies an FQDN or IP address. If it is a hostname, it resolves all
// A and AAAA records and ensures NO address resolves to internal or blocked subnets.
func (v *SSRFValidator) ValidateTarget(ctx context.Context, target string) (*HostValidationResult, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, errors.New("target cannot be empty")
	}

	// Strip port if present
	host := target
	if h, _, err := net.SplitHostPort(target); err == nil {
		host = h
	}

	// If direct IP address
	if ip := net.ParseIP(host); ip != nil {
		if blocked, reason := IsBlockedIP(ip); blocked {
			return nil, fmt.Errorf("SSRF blocked: target IP %s belongs to prohibited range: %s", ip.String(), reason)
		}
		return &HostValidationResult{
			Host:         host,
			ResolvedIPs:  []net.IP{ip},
			IPStrings:    []string{ip.String()},
			IsPublicOnly: true,
		}, nil
	}

	// Resolve hostname with strict context timeout
	resolveCtx, cancel := context.WithTimeout(ctx, v.timeout)
	defer cancel()

	ips, err := v.resolver.LookupIPAddr(resolveCtx, host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target hostname %q: %w", host, err)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("target hostname %q resolved to 0 IP addresses", host)
	}

	var validIPs []net.IP
	var ipStrings []string

	for _, addr := range ips {
		ip := addr.IP
		if blocked, reason := IsBlockedIP(ip); blocked {
			return nil, fmt.Errorf("SSRF blocked: hostname %q resolved to prohibited IP %s (%s)", host, ip.String(), reason)
		}
		validIPs = append(validIPs, ip)
		ipStrings = append(ipStrings, ip.String())
	}

	return &HostValidationResult{
		Host:         host,
		ResolvedIPs:  validIPs,
		IPStrings:    ipStrings,
		IsPublicOnly: true,
	}, nil
}
