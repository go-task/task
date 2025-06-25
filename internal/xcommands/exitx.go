package xcommands

import (
	"fmt"
	"os"
	"strconv"
)

// ExitxCommand implements the exitx command
type ExitxCommand struct{}

// Execute implements the Command interface
func (e *ExitxCommand) Execute(args []string, flags []string) error {
	exitCode := 0

	// Parse exit code from first argument if provided
	if len(args) > 0 {
		code, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("exitx: invalid exit code '%s': %w", args[0], err)
		}
		exitCode = code
	}

	// Exit with the specified code
	os.Exit(exitCode)
	return nil // This line will never be reached, but satisfies the interface
}

// Auto-register the command
func init() {
	Register("exitx", &ExitxCommand{})
}