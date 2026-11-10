package security

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"
)

// DNSVerifier manages DNS TXT ownership challenge generation and validation.
type DNSVerifier struct {
	resolver *net.Resolver
	timeout  time.Duration
}

// NewDNSVerifier creates a new verifier with a custom timeout.
func NewDNSVerifier(timeout time.Duration) *DNSVerifier {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &DNSVerifier{
		resolver: net.DefaultResolver,
		timeout:  timeout,
	}
}

// GenerateToken creates a high-entropy DNS verification challenge token.
func (v *DNSVerifier) GenerateToken() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random verification token: %w", err)
	}
	return "reconpilot-verification=" + hex.EncodeToString(bytes), nil
}

// VerifyOwnership queries DNS TXT records for rootDomain and verifies that the
// expected verification token is published. It checks both:
// 1. _reconpilot-challenge.<rootDomain>
// 2. <rootDomain>
func (v *DNSVerifier) VerifyOwnership(ctx context.Context, rootDomain, expectedToken string) (bool, string, error) {
	rootDomain = strings.TrimSpace(rootDomain)
	rootDomain = strings.TrimSuffix(rootDomain, ".")
	expectedToken = strings.TrimSpace(expectedToken)

	if rootDomain == "" {
		return false, "", fmt.Errorf("root domain cannot be empty")
	}
	if expectedToken == "" {
		return false, "", fmt.Errorf("expected token cannot be empty")
	}

	queryCtx, cancel := context.WithTimeout(ctx, v.timeout)
	defer cancel()

	candidates := []string{
		"_reconpilot-challenge." + rootDomain,
		rootDomain,
	}

	var allRecordsFound []string

	for _, host := range candidates {
		txtRecords, err := v.resolver.LookupTXT(queryCtx, host)
		if err != nil {
			// Continue to next candidate if not found
			continue
		}

		for _, record := range txtRecords {
			allRecordsFound = append(allRecordsFound, record)
			// Check exact match or token containment
			if strings.TrimSpace(record) == expectedToken || strings.Contains(record, expectedToken) {
				return true, fmt.Sprintf("Ownership verified via TXT record at %s", host), nil
			}
		}
	}

	summary := "No matching TXT record found."
	if len(allRecordsFound) > 0 {
		summary = fmt.Sprintf("Found %d TXT record(s), but none matched the challenge token %q.", len(allRecordsFound), expectedToken)
	}

	return false, summary, nil
}

