//go:build !windows

package codex

import (
	"os/exec"
	"syscall"
)

func configureProcess(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	// Put the codex process in its own process group so we can best-effort
	// terminate child processes on cancellation.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

func killProcessTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	pgid, err := syscall.Getpgid(pid)
	if err == nil && pgid > 0 {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		return
	}
	_ = cmd.Process.Kill()
}
