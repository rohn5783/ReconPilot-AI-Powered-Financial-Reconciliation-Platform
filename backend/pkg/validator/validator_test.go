package validator

import (
	"strings"
	"testing"
)

func TestValidateFQDN(t *testing.T) {
	validDomains := []string{
		"example.com",
		"sub.example.com",
		"dev-stage-01.infra.acme.org",
		"test.co.uk",
		"a.b.c.d.example.io",
	}

	for _, domain := range validDomains {
		if err := ValidateFQDN(domain); err != nil {
			t.Errorf("expected %q to be valid FQDN, got error: %v", domain, err)
		}
	}

	invalidDomains := []string{
		"",
		"   ",
		"localhost",
		"-badlabel.com",
		"badlabel-.com",
		"example..com",
		"example .com",
		"evil;command.com",
		"evil`whoami`.com",
		"http://example.com",
		strings.Repeat("a", 64) + ".com",
		strings.Repeat("a.", 130) + "com",
	}

	for _, domain := range invalidDomains {
		if err := ValidateFQDN(domain); err == nil {
			t.Errorf("expected %q to be invalid FQDN, but validation passed", domain)
		}
	}
}

func TestValidateCIDR(t *testing.T) {
	validCIDRs := []string{
		"192.168.1.0/24",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"1.1.1.1",
		"2001:db8::/32",
	}

	for _, cidr := range validCIDRs {
		netCIDR, err := ValidateCIDR(cidr)
		if err != nil {
			t.Errorf("expected CIDR %q to be valid, got: %v", cidr, err)
		}
		if netCIDR == nil {
			t.Errorf("expected non-nil netCIDR for %q", cidr)
		}
	}

	invalidCIDRs := []string{
		"",
		"not-an-ip",
		"999.999.999.999/24",
		"192.168.1.1/33",
	}

	for _, cidr := range invalidCIDRs {
		if _, err := ValidateCIDR(cidr); err == nil {
			t.Errorf("expected CIDR %q to fail validation, but it passed", cidr)
		}
	}
}

func TestValidateTargetURL(t *testing.T) {
	validURLs := []string{
		"https://example.com",
		"http://api.example.com:8080/health",
		"https://1.1.1.1:443",
	}

	for _, u := range validURLs {
		parsed, err := ValidateTargetURL(u)
		if err != nil {
			t.Errorf("expected URL %q to be valid, got: %v", u, err)
		}
		if parsed == nil {
			t.Errorf("expected non-nil parsed URL for %q", u)
		}
	}

	invalidURLs := []string{
		"",
		"ftp://example.com",
		"file:///etc/passwd",
		"https://admin:secret@example.com",
		"https://",
		"javascript:alert(1)",
	}

	for _, u := range invalidURLs {
		if _, err := ValidateTargetURL(u); err == nil {
			t.Errorf("expected URL %q to fail validation, but it passed", u)
		}
	}
}

