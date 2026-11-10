package security

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestIsBlockedIP(t *testing.T) {
	blockedCases := []struct {
		ip     string
		desc   string
	}{
		{"127.0.0.1", "IPv4 loopback"},
		{"127.0.1.1", "IPv4 loopback range"},
		{"10.0.0.1", "RFC 1918 Class A"},
		{"10.255.255.255", "RFC 1918 Class A upper"},
		{"172.16.0.1", "RFC 1918 Class B"},
		{"172.31.255.255", "RFC 1918 Class B upper"},
		{"192.168.1.1", "RFC 1918 Class C"},
		{"169.254.169.254", "AWS/GCP metadata link-local"},
		{"100.64.0.1", "Carrier-Grade NAT / CGNAT"},
		{"0.0.0.0", "Current network"},
		{"255.255.255.255", "Broadcast"},
		{"::1", "IPv6 loopback"},
		{"fc00::1", "IPv6 ULA"},
		{"fd12:3456:789a::1", "IPv6 ULA"},
		{"fe80::1", "IPv6 link-local"},
		{"::ffff:127.0.0.1", "IPv4-mapped IPv6 loopback"},
		{"::ffff:10.0.0.1", "IPv4-mapped IPv6 private"},
		{"::ffff:169.254.169.254", "IPv4-mapped IPv6 metadata"},
	}

	for _, tc := range blockedCases {
		ip := net.ParseIP(tc.ip)
		if ip == nil {
			t.Fatalf("failed to parse IP %q", tc.ip)
		}
		blocked, reason := IsBlockedIP(ip)
		if !blocked {
			t.Errorf("expected IP %s (%s) to be BLOCKED, but it was allowed", tc.ip, tc.desc)
		}
		if reason == "" {
			t.Errorf("expected non-empty block reason for IP %s", tc.ip)
		}
	}

	allowedCases := []struct {
		ip   string
		desc string
	}{
		{"1.1.1.1", "Cloudflare DNS public"},
		{"8.8.8.8", "Google DNS public"},
		{"93.184.216.34", "Example.com public IP"},
		{"2606:4700:4700::1111", "Cloudflare public IPv6"},
		{"2001:4860:4860::8888", "Google public IPv6"},
	}

	for _, tc := range allowedCases {
		ip := net.ParseIP(tc.ip)
		if ip == nil {
			t.Fatalf("failed to parse IP %q", tc.ip)
		}
		blocked, reason := IsBlockedIP(ip)
		if blocked {
			t.Errorf("expected IP %s (%s) to be ALLOWED, but it was blocked with reason: %s", tc.ip, tc.desc, reason)
		}
	}
}

func TestSSRFValidatorDirectIP(t *testing.T) {
	v := NewSSRFValidator(2 * time.Second)

	// Blocked target
	_, err := v.ValidateTarget(context.Background(), "127.0.0.1")
	if err == nil {
		t.Errorf("expected SSRF block error for 127.0.0.1, got nil")
	}

	// Metadata target
	_, err = v.ValidateTarget(context.Background(), "169.254.169.254:80")
	if err == nil {
		t.Errorf("expected SSRF block error for cloud metadata target, got nil")
	}

	// Public IP target
	res, err := v.ValidateTarget(context.Background(), "1.1.1.1")
	if err != nil {
		t.Errorf("expected 1.1.1.1 to pass SSRF validation, got error: %v", err)
	}
	if res == nil || len(res.ResolvedIPs) == 0 {
		t.Errorf("expected non-empty resolved IPs for 1.1.1.1")
	}
}

