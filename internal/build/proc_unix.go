//go:build !windows

package build

import (
	"os/exec"
	"syscall"
)

// setProcessGroup runs cmd in its own process group and makes cancellation
// kill the whole group, so children of the build tool die with it.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
