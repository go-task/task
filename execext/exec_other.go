// +build !windows

package execext

import (
	"os/exec"
)

// NewCommand returns a new command that runs on "sh" is available or on "cmd"
// otherwise on Windows
func NewCommand(c string) *exec.Cmd {
	return newShCommand(c)
}
