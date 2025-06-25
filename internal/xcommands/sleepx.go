package xcommands

import (
	"fmt"
	"strconv"
	"time"
)

// SleepxCommand implements the sleepx command (delay execution)
type SleepxCommand struct{}

// Execute implements the Command interface
func (s *SleepxCommand) Execute(args []string, flags []string) error {
	if len(args) == 0 {
		return fmt.Errorf("sleepx: missing operand")
	}

	if len(args) > 1 {
		return fmt.Errorf("sleepx: too many arguments")
	}

	durationStr := args[0]
	
	// Parse the duration
	duration, err := parseDuration(durationStr)
	if err != nil {
		return fmt.Errorf("sleepx: invalid time interval '%s': %w", durationStr, err)
	}

	if duration < 0 {
		return fmt.Errorf("sleepx: invalid time interval '%s': negative duration", durationStr)
	}

	// Sleep for the specified duration
	time.Sleep(duration)
	return nil
}

// parseDuration parses various duration formats
// Supports: "1" (seconds), "0.5" (fractional seconds), "1m" (minutes), "30s" (seconds)
func parseDuration(s string) (time.Duration, error) {
	// First try parsing as a Go duration (e.g., "1m", "30s", "1h30m")
	if duration, err := time.ParseDuration(s); err == nil {
		return duration, nil
	}

	// If that fails, try parsing as a plain number (seconds)
	if seconds, err := strconv.ParseFloat(s, 64); err == nil {
		// Convert seconds to duration
		return time.Duration(seconds * float64(time.Second)), nil
	}

	return 0, fmt.Errorf("invalid duration format")
}

// Auto-register the command
func init() {
	Register("sleepx", &SleepxCommand{})
}