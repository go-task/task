package xcommands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// XargsxCommand implements the xargsx command for building and executing commands from stdin
type XargsxCommand struct{}

// Execute implements the Command interface
func (x *XargsxCommand) Execute(args []string, flags []string) error {
	if len(args) == 0 {
		return fmt.Errorf("xargsx: missing command to execute")
	}

	// Read input from stdin
	scanner := bufio.NewScanner(os.Stdin)
	var inputArgs []string

	// Read all lines from stdin and split on whitespace
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			// Split the line on whitespace and append to inputArgs
			fields := strings.Fields(line)
			inputArgs = append(inputArgs, fields...)
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("xargsx: error reading from stdin: %w", err)
	}

	// Build the command with the provided arguments and stdin arguments
	command := args[0]
	commandArgs := append(args[1:], inputArgs...)

	// Execute the command
	cmd := exec.Command(command, commandArgs...)
	
	// Connect stdout and stderr to the current process
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Execute the command
	if err := cmd.Run(); err != nil {
		// Handle different types of exec errors
		if exitError, ok := err.(*exec.ExitError); ok {
			// Command executed but returned non-zero exit code
			return fmt.Errorf("xargsx: command '%s' exited with code %d", command, exitError.ExitCode())
		}
		// Other execution errors (command not found, etc.)
		return fmt.Errorf("xargsx: failed to execute '%s': %w", command, err)
	}

	return nil
}

// Auto-register the command
func init() {
	Register("xargsx", &XargsxCommand{})
}