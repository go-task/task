// +build !windows

package execext

import (
	"context"
	"os/exec"
)

// NewCommand returns a new command that runs on "sh" is available or on "cmd"
// otherwise on Windows
func NewCommand(ctx context.Context, c string) *exec.Cmd {
	return newShCommand(ctx, c)
}
