package xcommands

import (
	"fmt"
	"strings"
)

// EchoxCommand implements the echox command
type EchoxCommand struct{}

// Execute implements the Command interface
func (e *EchoxCommand) Execute(args []string, flags []string) error {
	// Handle flags (dashes already stripped)
	suppressNewline := false
	for _, flag := range flags {
		switch flag {
		case "n":
			suppressNewline = true
		}
	}

	// Join arguments with spaces
	output := strings.Join(args, " ")

	// Print output with or without newline based on -n flag
	if suppressNewline {
		fmt.Print(output)
	} else {
		fmt.Println(output)
	}

	return nil
}

// Auto-register the command
func init() {
	Register("echox", &EchoxCommand{})
}