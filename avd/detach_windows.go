//go:build windows

package avd

import (
	"os/exec"
	"syscall"
)

const createNewProcessGroup = 0x00000008

// detach puts the emulator in its own process group so closing gadb does not
// take the emulator down with it.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNewProcessGroup}
}
