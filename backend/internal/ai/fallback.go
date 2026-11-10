package ai

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"reconpilot/internal/domain"
)

// FallbackGenerator produces deterministic, high-fidelity security briefings
// when the Google GenAI API is not configured, rate-limited, or unavailable.
type FallbackGenerator struct{}

// NewFallbackGenerator creates a new FallbackGenerator.
func NewFallbackGenerator() *FallbackGenerator {
	return &FallbackGenerator{}
}

// Generate creates a structured AI report from the compacted scan payload.
func (g *FallbackGenerator) Generate(scanJobID uuid.UUID, payload *CompactPayload) *domain.AIReport {
	// 1. Calculate Risk Score (0-100)
	score := 15 // Base baseline score for exposed attack surface
	score += payload.SeverityStats["critical"] * 30
	score += payload.SeverityStats["high"] * 15
	score += payload.SeverityStats["medium"] * 5
	score += payload.SeverityStats["low"] * 2

	// Check for sensitive open ports
	for _, a := range payload.Assets {
		for _, p := range a.OpenPorts {
			if p == 6379 || p == 3306 || p == 5432 || p == 27017 || p == 9200 {
				score += 15
			} else if p == 22 || p == 8200 {
				score += 5
			}
		}
	}

	if score > 100 {
		score = 100
	}
	if score < 10 && payload.TotalAssets > 0 {
		score = 10
	}

	// 2. Synthesize Executive Summary
	var posture string
	if score >= 75 {
		posture = "CRITICAL ELEVATED RISK"
	} else if score >= 50 {
		posture = "HIGH ATTENTION REQUIRED"
	} else if score >= 25 {
		posture = "MODERATE DEFENSIVE HYGIENE NEEDED"
	} else {
		posture = "NOMINAL RESILIENT POSTURE"
	}

	execSummary := fmt.Sprintf(
		"External Attack Surface Assessment for %s established an overall posture of %s (Risk Score: %d/100). "+
			"Reconnaissance across %d perimeter assets identified %d critical, %d high, %d medium, and %d low-severity findings. ",
		payload.TargetDomain, posture, score, payload.TotalAssets,
		payload.SeverityStats["critical"], payload.SeverityStats["high"],
		payload.SeverityStats["medium"], payload.SeverityStats["low"],
	)

	if payload.DriftSummary != nil && len(payload.DriftSummary.HighRiskMutations) > 0 {
		execSummary += fmt.Sprintf("Critical historical drift detected: %s. ",
			strings.Join(payload.DriftSummary.HighRiskMutations, "; "))
	} else {
		execSummary += "Immediate defensive priorities focus on eliminating unauthenticated administrative services and tightening perimeter ingress access controls."
	}

	// 3. Prioritized Findings
	var prioritized []domain.PrioritizedFinding
	for i, f := range payload.Findings {
		affected := strings.Join(f.AffectedHosts, ", ")
		if affected == "" {
			affected = payload.TargetDomain
		}

		exploitProb := "Opportunistic / Medium"
		impact := "Information disclosure and architectural reconnaissance."
		rootCause := "Insecure default configuration."

		if f.Severity == domain.SeverityCritical {
			exploitProb = "Active / Trivial Remote Exploit"
			impact = "Arbitrary code execution or full service database compromise."
			rootCause = "Exposed unauthenticated cache or data store directly on public interface."
		} else if f.Severity == domain.SeverityHigh {
			exploitProb = "High / Public PoC Accessible"
			impact = "Sensitive source code, credential leak, or administrative feature takeover."
			rootCause = "Missing endpoint authentication or exposed VCS metadata directory."
		} else if f.Severity == domain.SeverityMedium {
			exploitProb = "Moderate / Multi-step Chain"
			impact = "Unauthorized cross-origin resource requests or session eavesdropping."
			rootCause = "Permissive CORS policies or legacy TLS cipher suite enablement."
		}

		prioritized = append(prioritized, domain.PrioritizedFinding{
			Rank:               i + 1,
			Title:              f.Title,
			Severity:           f.Severity,
			AffectedAsset:      affected,
			ExploitProbability: exploitProb,
			ImpactAnalysis:     impact,
			RootCause:          rootCause,
		})
	}

	// 4. Remediation Playbook
	var playbook []domain.RemediationItem

	// Generate tailored remediation items matching findings
	for _, f := range payload.Findings {
		switch {
		case strings.Contains(strings.ToLower(f.RuleID), "redis"):
			playbook = append(playbook, domain.RemediationItem{
				Title:          "Harden & Isolate Exposed Redis Cache",
				TargetCategory: "AWS_SG / Linux Firewall",
				Severity:       string(domain.SeverityCritical),
				ConfigSnippet: `# /etc/redis/redis.conf
bind 127.0.0.1 ::1
protected-mode yes
requirepass "SECURE_RANDOMLY_GENERATED_64_CHAR_TOKEN"

# Immediate AWS CLI security group revocation:
aws ec2 revoke-security-group-ingress \
  --group-id sg-0123456789abcdef0 \
  --protocol tcp --port 6379 \
  --cidr 0.0.0.0/0`,
				Explanation: "Binds Redis strictly to loopback interfaces, enforces strong authentication, and removes 0.0.0.0/0 ingress rules from cloud security groups.",
				ValidationSteps: []string{
					"Run `nc -zv <target> 6379` from an external network; verify connection times out or is refused.",
					"Verify local clients authenticate successfully with `AUTH <password>`.",
				},
			})

		case strings.Contains(strings.ToLower(f.RuleID), "git"):
			playbook = append(playbook, domain.RemediationItem{
				Title:          "Block Public Access to .git Directories",
				TargetCategory: "Nginx",
				Severity:       string(domain.SeverityHigh),
				ConfigSnippet: `# /etc/nginx/conf.d/security.conf
location ~ /\.git {
    deny all;
    return 404;
}

# Alternatively in Apache httpd.conf:
# <Directorymatch "^/.*/\.git/">
#    Order deny,allow
#    Deny from all
# </Directorymatch>`,
				Explanation: "Prevents attackers from downloading the .git directory structure and recovering proprietary source code, secrets, and internal commit histories.",
				ValidationSteps: []string{
					"Run `curl -isk 'https://<target>/.git/config'`; verify HTTP 404 response.",
					"Verify deployment scripts do not deploy .git metadata to production webroots.",
				},
			})

		case strings.Contains(strings.ToLower(f.RuleID), "actuator"):
			playbook = append(playbook, domain.RemediationItem{
				Title:          "Restrict Spring Boot Actuator Diagnostic Endpoints",
				TargetCategory: "Spring Boot / Docker",
				Severity:       string(domain.SeverityHigh),
				ConfigSnippet: `# application.properties (or application.yml)
management.endpoints.web.exposure.include=health,info
management.endpoints.web.exposure.exclude=env,beans,heapdump,mappings,threaddump
management.endpoint.health.show-details=never`,
				Explanation: "Restricts exposed Spring Actuator endpoints to only basic health checks, preventing environment variable leaks and heap dump extractions.",
				ValidationSteps: []string{
					"Run `curl -isk 'https://<target>/actuator/env'`; verify HTTP 404 or 401 response.",
				},
			})

		case strings.Contains(strings.ToLower(f.RuleID), "cors"):
			playbook = append(playbook, domain.RemediationItem{
				Title:          "Enforce Explicit CORS Origin Whitelist",
				TargetCategory: "Nginx / API Gateway",
				Severity:       string(domain.SeverityMedium),
				ConfigSnippet: `# Nginx map for strict origin whitelisting
map $http_origin $cors_origin {
    default "";
    "~^https://(dashboard|portal)\.yourdomain\.com$" "$http_origin";
}

server {
    location /api/ {
        if ($cors_origin != "") {
            add_header Access-Control-Allow-Origin $cors_origin always;
            add_header Access-Control-Allow-Credentials "true" always;
            add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS" always;
            add_header Access-Control-Allow-Headers "Authorization, Content-Type" always;
        }
    }
}`,
				Explanation: "Stops reflecting arbitrary request origins and prevents credentialed cross-origin API exploitation.",
				ValidationSteps: []string{
					"Send request with `Origin: https://evil.com`; verify Access-Control-Allow-Origin is absent.",
					"Send request with legitimate origin; verify header matches whitelist.",
				},
			})

		case strings.Contains(strings.ToLower(f.RuleID), "tls"):
			playbook = append(playbook, domain.RemediationItem{
				Title:          "Deprecate Weak TLS 1.0/1.1 and CBC Ciphers",
				TargetCategory: "TLS / Nginx / Cloudflare",
				Severity:       string(domain.SeverityMedium),
				ConfigSnippet: `# Nginx Modern TLS Configuration (Mozilla SSL Configuration Generator)
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384';
ssl_prefer_server_ciphers off;
ssl_session_timeout 1d;
ssl_session_cache shared:SSL:10m;`,
				Explanation: "Enforces AEAD ciphers only and removes legacy CBC ciphers susceptible to padding oracle attacks.",
				ValidationSteps: []string{
					"Run `openssl s_client -connect <target>:443 -tls1_1`; verify handshake failure.",
					"Run `testssl.sh <target>` to verify Grade A TLS posture.",
				},
			})

		case strings.Contains(strings.ToLower(f.RuleID), "hsts"):
			playbook = append(playbook, domain.RemediationItem{
				Title:          "Enable HTTP Strict Transport Security (HSTS)",
				TargetCategory: "Nginx / Cloudflare",
				Severity:       string(domain.SeverityLow),
				ConfigSnippet: `# Enforce HSTS across all subdomains with 2-year duration and preload
add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;`,
				Explanation: "Instructs browsers to never communicate over cleartext HTTP, preventing SSL stripping downgrade attacks.",
				ValidationSteps: []string{
					"Run `curl -sI 'https://<target>' | grep -i Strict-Transport-Security`.",
					"Confirm header contains `max-age=63072000; includeSubDomains; preload`.",
				},
			})
		}
	}

	// Always ensure at least one general perimeter hardening item exists if playbook is empty
	if len(playbook) == 0 {
		playbook = append(playbook, domain.RemediationItem{
			Title:          "Perimeter Ingress Firewall Hardening",
			TargetCategory: "Firewall",
			Severity:       string(domain.SeverityLow),
			ConfigSnippet: `# Ensure only HTTPS (443) and HTTP (80) are open externally:
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable`,
			Explanation: "Applies least-privilege perimeter ingress filtering.",
			ValidationSteps: []string{
				"Run `nmap -Pn <target>` and verify only 80 and 443 are listening.",
			},
		})
	}

	driftAnalysis := domain.DriftAnalysis{}
	if payload.DriftSummary != nil {
		driftAnalysis = *payload.DriftSummary
	}

	return &domain.AIReport{
		ID:                         uuid.New(),
		ScanJobID:                  scanJobID,
		ExecutiveSummary:           execSummary,
		OverallRiskScore:           score,
		AttackSurfaceDriftAnalysis: driftAnalysis,
		PrioritizedFindings:        prioritized,
		RemediationPlaybook:        playbook,
		TokensUsed:                 0,
		ModelVersion:               "rule-engine-fallback-v1",
		IsFallback:                 true,
		GeneratedAt:                time.Now(),
	}
}

