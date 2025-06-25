package xcommands

import (
	"fmt"
	"os"
)

// UnsetxCommand implements the unsetx command for unsetting environment variables
type UnsetxCommand struct{}

// Execute implements the Command interface
func (u *UnsetxCommand) Execute(args []string, flags []string) error {
	if len(args) == 0 {
		return fmt.Errorf("unsetx: missing operands")
	}

	// Unset each specified environment variable
	for _, varName := range args {
		if varName == "" {
			continue // Skip empty variable names
		}
		
		// Check if the variable exists (optional, for informative purposes)
		_, exists := os.LookupEnv(varName)
		if !exists {
			// Variable doesn't exist, but this is not an error condition
			// Some shells/systems treat this as a no-op
			continue
		}
		
		// Unset the environment variable
		if err := os.Unsetenv(varName); err != nil {
			return fmt.Errorf("unsetx: cannot unset '%s': %w", varName, err)
		}
	}

	return nil
}

// Auto-register the command
func init() {
	Register("unsetx", &UnsetxCommand{})
}