//go:build windows

package build

import "os/exec"

// setProcessGroup is a no-op on Windows; cancellation kills the process only.
func setProcessGroup(*exec.Cmd) {}
