//go:build windows

package runner

import (
	"os/exec"
	"strconv"
)

// prepareCommandSysProcAttr sets Windows-specific process attributes.
func prepareCommandSysProcAttr(cmd *exec.Cmd) {
	// On Windows, child processes are handled via process trees.
}

// killProcessGroup terminates the entire Windows process tree using taskkill.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil || cmd.Process.Pid <= 0 {
		return nil
	}

	pidStr := strconv.Itoa(cmd.Process.Pid)
	killCmd := exec.Command("taskkill", "/T", "/F", "/PID", pidStr)
	if err := killCmd.Run(); err != nil {
		// Fallback to direct process kill
		return cmd.Process.Kill()
	}
	return nil
}
