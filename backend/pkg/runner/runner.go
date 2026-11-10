package runner

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ExecutionConfig defines parameters for a hardened, bounded subprocess run.
type ExecutionConfig struct {
	Binary         string
	Args           []string
	Timeout        time.Duration
	Env            []string
	Dir            string
	RingBufferSize int
	OnStdoutLine   func(line string)
	OnStderrLine   func(line string)
	Stdin          io.Reader
}

// ExecutionResult captures the output, metrics, and completion state of the process.
type ExecutionResult struct {
	ExitCode         int
	Duration         time.Duration
	Stdout           []byte
	Stderr           []byte
	TotalStdoutBytes int64
	TotalStderrBytes int64
	KilledByTimeout  bool
	Error            error
}

// SafeRunner provides a hardened subprocess execution engine preventing command injection,
// memory exhaustion (OOM), and orphaned zombie processes.
type SafeRunner struct {
	defaultTimeout time.Duration
	defaultRingKB  int
}

// NewSafeRunner creates a new SafeRunner instance.
func NewSafeRunner(defaultTimeout time.Duration, defaultRingKB int) *SafeRunner {
	if defaultTimeout <= 0 {
		defaultTimeout = 15 * time.Minute
	}
	if defaultRingKB <= 0 {
		defaultRingKB = 1024 // 1 MB
	}
	return &SafeRunner{
		defaultTimeout: defaultTimeout,
		defaultRingKB:  defaultRingKB,
	}
}

// ValidateBinary checks whether the requested executable exists in PATH or at the specified path.
func (r *SafeRunner) ValidateBinary(binName string) (string, error) {
	binName = strings.TrimSpace(binName)
	if binName == "" {
		return "", errors.New("binary name or path cannot be empty")
	}

	// Reject attempts to inject shell meta-characters into binary path
	if strings.ContainsAny(binName, ";|&$`><\n\r") {
		return "", fmt.Errorf("dangerous characters detected in binary path: %q", binName)
	}

	path, err := exec.LookPath(binName)
	if err != nil {
		return "", fmt.Errorf("binary %q not found in system PATH: %w", binName, err)
	}

	return path, nil
}

// Run executes a binary strictly through exec.CommandContext without shell invocation.
// It enforces process group cleanup, memory-bounded ring buffers, and line telemetry streaming.
func (r *SafeRunner) Run(ctx context.Context, cfg ExecutionConfig) (*ExecutionResult, error) {
	binPath, err := r.ValidateBinary(cfg.Binary)
	if err != nil {
		return nil, err
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = r.defaultTimeout
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, binPath, cfg.Args...)

	// OS-specific process group isolation
	prepareCommandSysProcAttr(cmd)

	// Directory and environment sanitization
	if cfg.Dir != "" {
		cmd.Dir = cfg.Dir
	}

	// Retain only safe environment variables; prevent shell hijack
	cleanEnv := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"USERPROFILE=" + os.Getenv("USERPROFILE"),
		"TMP=" + os.Getenv("TMP"),
		"TEMP=" + os.Getenv("TEMP"),
	}
	if len(cfg.Env) > 0 {
		cleanEnv = append(cleanEnv, cfg.Env...)
	}
	cmd.Env = cleanEnv

	ringCap := cfg.RingBufferSize
	if ringCap <= 0 {
		ringCap = r.defaultRingKB * 1024
	}

	stdoutRing := NewRingBuffer(ringCap)
	stderrRing := NewRingBuffer(ringCap)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if cfg.Stdin != nil {
		cmd.Stdin = cfg.Stdin
	}

	startTime := time.Now()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process %q: %w", cfg.Binary, err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Stream stdout with custom buffer to handle long lines (up to 1MB)
	go func() {
		defer wg.Done()
		reader := io.TeeReader(stdoutPipe, stdoutRing)
		scanner := bufio.NewScanner(reader)
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if cfg.OnStdoutLine != nil {
				cfg.OnStdoutLine(line)
			}
		}
	}()

	// Stream stderr
	go func() {
		defer wg.Done()
		reader := io.TeeReader(stderrPipe, stderrRing)
		scanner := bufio.NewScanner(reader)
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if cfg.OnStderrLine != nil {
				cfg.OnStderrLine(line)
			}
		}
	}()

	// Wait for readers to finish reading pipes
	wg.Wait()

	cmdErr := cmd.Wait()
	duration := time.Since(startTime)

	result := &ExecutionResult{
		Duration:         duration,
		Stdout:           stdoutRing.Bytes(),
		Stderr:           stderrRing.Bytes(),
		TotalStdoutBytes: stdoutRing.TotalBytesWritten(),
		TotalStderrBytes: stderrRing.TotalBytesWritten(),
		ExitCode:         0,
		Error:            cmdErr,
	}

	if execCtx.Err() == context.DeadlineExceeded {
		result.KilledByTimeout = true
		_ = killProcessGroup(cmd)
		result.Error = fmt.Errorf("process execution exceeded timeout of %v: %w", timeout, context.DeadlineExceeded)
		result.ExitCode = -1
		return result, result.Error
	}

	if cmdErr != nil {
		var exitErr *exec.ExitError
		if errors.As(cmdErr, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	}

	return result, cmdErr
}

