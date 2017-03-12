package execext

import (
	"os/exec"
)

var (
	// ShPath is path to "sh" command
	ShPath string
	// ShExists is true if "sh" command is available on the system
	ShExists bool
)

func init() {
	var err error
	ShPath, err = exec.LookPath("sh")
	ShExists = err == nil
}

func newShCommand(c string) *exec.Cmd {
	return exec.Command(ShPath, "-c", c)
}
