//go:build !windows

package avd

import (
	"os/exec"
	"syscall"
)

// detach puts the emulator in its own session so closing gadb or the terminal
// does not take the emulator down with it.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
