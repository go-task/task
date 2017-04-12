package execext

import (
	"context"
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

func newShCommand(ctx context.Context, c string) *exec.Cmd {
	return exec.CommandContext(ctx, ShPath, "-c", c)
}
