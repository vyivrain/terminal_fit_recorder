//go:build !windows

package trainer

import (
	"os/exec"
	"syscall"
)

// detach starts the child in its own session, so Ctrl-C in the terminal or
// closing it does not kill a review still in progress.
func detach(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
