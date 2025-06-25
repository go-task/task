package xcommands

import (
	"fmt"
	"os"
)

// MkdirxCommand implements the mkdirx command
type MkdirxCommand struct{}

// Execute implements the Command interface
func (m *MkdirxCommand) Execute(args []string, flags []string) error {
	if len(args) == 0 {
		return fmt.Errorf("mkdirx: missing operands")
	}

	// Handle flags (dashes already stripped)
	createParents := false
	for _, flag := range flags {
		switch flag {
		case "p":
			createParents = true
		default:
			return fmt.Errorf("mkdirx: invalid option -- '%s'", flag)
		}
	}

	// Create each directory specified
	for _, dir := range args {
		if err := createDirectory(dir, createParents); err != nil {
			return err
		}
	}

	return nil
}

// createDirectory handles the actual directory creation logic
func createDirectory(dir string, createParents bool) error {
	if createParents {
		// Create directory and all necessary parent directories
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdirx: cannot create directory '%s': %w", dir, err)
		}
	} else {
		// Create only the specified directory (parent must exist)
		if err := os.Mkdir(dir, 0755); err != nil {
			if os.IsExist(err) {
				return fmt.Errorf("mkdirx: cannot create directory '%s': File exists", dir)
			}
			if os.IsNotExist(err) {
				return fmt.Errorf("mkdirx: cannot create directory '%s': No such file or directory", dir)
			}
			return fmt.Errorf("mkdirx: cannot create directory '%s': %w", dir, err)
		}
	}

	return nil
}

// Auto-register the command
func init() {
	Register("mkdirx", &MkdirxCommand{})
}