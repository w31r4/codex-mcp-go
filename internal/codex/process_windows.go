//go:build windows

package codex

import "os/exec"

func configureProcess(cmd *exec.Cmd) {
	// No-op for now.
}

func killProcessTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
