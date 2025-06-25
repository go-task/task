package xcommands

import (
	"fmt"
	"os"
)

// PwdxCommand implements the pwdx command (print working directory)
type PwdxCommand struct{}

// Execute implements the Command interface
func (p *PwdxCommand) Execute(args []string, flags []string) error {
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("pwdx: cannot get current directory: %w", err)
	}

	// Print the current working directory
	fmt.Println(wd)
	return nil
}

// Auto-register the command
func init() {
	Register("pwdx", &PwdxCommand{})
}