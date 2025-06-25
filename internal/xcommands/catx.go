package xcommands

import (
	"fmt"
	"io"
	"os"
)

// CatxCommand implements the catx command
type CatxCommand struct{}

// Execute implements the Command interface
func (c *CatxCommand) Execute(args []string, flags []string) error {
	// If no arguments provided, read from stdin
	if len(args) == 0 {
		return c.copyToStdout(os.Stdin, "stdin")
	}

	// Process each file argument
	for _, filename := range args {
		if err := c.processFile(filename); err != nil {
			return err
		}
	}

	return nil
}

// processFile opens and processes a single file
func (c *CatxCommand) processFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("catx: %s: %w", filename, err)
	}
	defer file.Close()

	return c.copyToStdout(file, filename)
}

// copyToStdout copies content from reader to stdout
func (c *CatxCommand) copyToStdout(reader io.Reader, source string) error {
	_, err := io.Copy(os.Stdout, reader)
	if err != nil {
		return fmt.Errorf("catx: error reading from %s: %w", source, err)
	}
	return nil
}

// Auto-register the command
func init() {
	Register("catx", &CatxCommand{})
}