package trainer

import "os/exec"

// detach is a no-op on Windows, where a started child already survives its
// parent exiting.
func detach(*exec.Cmd) {}
