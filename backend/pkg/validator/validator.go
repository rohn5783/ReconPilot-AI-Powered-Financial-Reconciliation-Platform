package validator

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

var (
	// fqdnRegex matches standard hostnames and FQDNs per RFC 1123.
	// Labels must be 1-63 chars, alphanum or hyphen, not starting or ending with hyphen.
	// Minimum 2 labels for an FQDN.
	fqdnRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,63}$`)

	// singleLabelRegex matches a single hostname label.
	labelRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`)
)

// ValidateFQDN checks if a domain name conforms strictly to RFC 1123 and length standards.
func ValidateFQDN(fqdn string) error {
	fqdn = strings.TrimSpace(fqdn)
	if fqdn == "" {
		return errors.New("domain name cannot be empty")
	}

	// Remove trailing dot if present (FQDN absolute notation)
	fqdn = strings.TrimSuffix(fqdn, ".")

	if len(fqdn) > 253 {
		return fmt.Errorf("domain name exceeds maximum RFC length of 253 characters (got %d)", len(fqdn))
	}

	// Reject if domain contains disallowed characters or control characters
	for _, ch := range fqdn {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '-' && ch != '.' {
			return fmt.Errorf("domain name contains invalid character %q", ch)
		}
	}

	if !fqdnRegex.MatchString(fqdn) {
		return fmt.Errorf("invalid FQDN format: %q", fqdn)
	}

	// Validate individual label lengths
	labels := strings.Split(fqdn, ".")
	if len(labels) < 2 {
		return fmt.Errorf("domain must have at least one subdomain/domain and a valid TLD: %q", fqdn)
	}

	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return fmt.Errorf("domain label %q has invalid length (must be 1-63 chars)", label)
		}
		if !labelRegex.MatchString(label) {
			return fmt.Errorf("domain label %q contains invalid characters or starts/ends with hyphen", label)
		}
	}

	return nil
}

// ValidateCIDR verifies an IP address or CIDR notation.
func ValidateCIDR(cidrStr string) (*net.IPNet, error) {
	cidrStr = strings.TrimSpace(cidrStr)
	if cidrStr == "" {
		return nil, errors.New("CIDR string cannot be empty")
	}

	// If no slash, treat as single host /32 or /128
	if !strings.Contains(cidrStr, "/") {
		ip := net.ParseIP(cidrStr)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP address: %q", cidrStr)
		}
		if ip.To4() != nil {
			cidrStr += "/32"
		} else {
			cidrStr += "/128"
		}
	}

	_, ipNet, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR block %q: %w", cidrStr, err)
	}

	return ipNet, nil
}

// ValidateTargetURL validates target URLs for live probing, rejecting credentials and dangerous schemes.
func ValidateTargetURL(rawURL string) (*url.URL, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, errors.New("URL cannot be empty")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme %q (only http and https allowed)", parsed.Scheme)
	}

	if parsed.User != nil {
		return nil, errors.New("URLs containing embedded user credentials are not permitted")
	}

	host := parsed.Hostname()
	if host == "" {
		return nil, errors.New("URL must specify a non-empty hostname")
	}

	// Check if hostname is an IP or FQDN
	if ip := net.ParseIP(host); ip == nil {
		if err := ValidateFQDN(host); err != nil {
			return nil, fmt.Errorf("invalid hostname in URL: %w", err)
		}
	}

	return parsed, nil
}

