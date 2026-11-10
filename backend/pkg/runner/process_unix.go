//go:build !windows

package runner

import (
	"os/exec"
	"syscall"
)

// prepareCommandSysProcAttr sets POSIX process group attributes so all sub-processes
// can be killed atomically via their negative process group ID.
func prepareCommandSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

// killProcessGroup terminates the entire process group with SIGKILL.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil || cmd.Process.Pid <= 0 {
		return nil
	}
	// Negative PID targets the entire process group in POSIX
	pgid := cmd.Process.Pid
	return syscall.Kill(-pgid, syscall.SIGKILL)
}

