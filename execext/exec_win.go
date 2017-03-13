// +build windows

package execext

import (
	"os/exec"
)

// NewCommand returns a new command that runs on "sh" is available or on "cmd"
// otherwise on Windows
func NewCommand(c string) *exec.Cmd {
	if ShExists {
		return newShCommand(c)
	}
	return newCmdCommand(c)
}

func newCmdCommand(c string) *exec.Cmd {
	return exec.Command("cmd", "/C", c)
}
