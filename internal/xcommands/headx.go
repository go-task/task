package xcommands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
)

// HeadxCommand implements the headx command
type HeadxCommand struct{}

// Execute implements the Command interface
func (h *HeadxCommand) Execute(args []string, flags []string) error {
	// Default number of lines to display
	lines := 10

	// Parse -n flag for number of lines
	for i, flag := range flags {
		if flag == "n" {
			// Next flag should be the number
			if i+1 < len(flags) {
				num, err := strconv.Atoi(flags[i+1])
				if err != nil {
					return fmt.Errorf("headx: invalid number of lines '%s': %w", flags[i+1], err)
				}
				if num < 0 {
					return fmt.Errorf("headx: number of lines cannot be negative: %d", num)
				}
				lines = num
				break
			} else {
				return fmt.Errorf("headx: option -n requires an argument")
			}
		}
		// Handle combined -n<number> format (e.g., -n20)
		if len(flag) > 1 && flag[0] == 'n' {
			num, err := strconv.Atoi(flag[1:])
			if err != nil {
				return fmt.Errorf("headx: invalid number of lines '%s': %w", flag[1:], err)
			}
			if num < 0 {
				return fmt.Errorf("headx: number of lines cannot be negative: %d", num)
			}
			lines = num
			break
		}
	}

	// If no arguments, read from stdin
	if len(args) == 0 {
		return h.processFile(os.Stdin, "", lines, false)
	}

	// Process each file
	for i, filename := range args {
		file, err := os.Open(filename)
		if err != nil {
			return fmt.Errorf("headx: cannot open '%s': %w", filename, err)
		}

		// Show header for multiple files
		showHeader := len(args) > 1
		if showHeader && i > 0 {
			fmt.Println() // Add blank line between files
		}

		err = h.processFile(file, filename, lines, showHeader)
		file.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// processFile reads and displays the first n lines from a file
func (h *HeadxCommand) processFile(file io.Reader, filename string, lines int, showHeader bool) error {
	if showHeader {
		fmt.Printf("==> %s <==\n", filename)
	}

	scanner := bufio.NewScanner(file)
	lineCount := 0

	for scanner.Scan() && lineCount < lines {
		fmt.Println(scanner.Text())
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		if filename != "" {
			return fmt.Errorf("headx: error reading '%s': %w", filename, err)
		}
		return fmt.Errorf("headx: error reading from stdin: %w", err)
	}

	return nil
}

// Auto-register the command
func init() {
	Register("headx", &HeadxCommand{})
}