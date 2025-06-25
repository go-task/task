package xcommands

import (
	"context"
	"strings"
	"sync"

	"mvdan.cc/sh/v3/interp"
)

// Command interface that all cross-platform commands must implement
type Command interface {
	Execute(args []string, flags []string) error
}

// Registry holds all registered commands
var (
	registry = make(map[string]Command)
	mu       sync.RWMutex
)

// Register allows commands to register themselves
func Register(name string, cmd Command) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = cmd
}

// ExecHandler is the integration point with execext
func ExecHandler(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(ctx context.Context, args []string) error {
		if len(args) == 0 {
			return next(ctx, args)
		}

		command := args[0]
		rawArgs := args[1:]

		// Check if we have a registered coreutils command
		mu.RLock()
		cmdImpl, exists := registry[command]
		mu.RUnlock()

		if exists {
			flags, cleanArgs := parseFlags(rawArgs)
			return cmdImpl.Execute(cleanArgs, flags)
		}

		// Fall back to default shell execution
		return next(ctx, args)
	}
}

// parseFlags separates flags (starting with -) from regular arguments
// Strips the leading dash(es) from flags
func parseFlags(rawArgs []string) (flags []string, args []string) {
	for _, arg := range rawArgs {
		if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			// Strip leading dashes
			flag := strings.TrimLeft(arg, "-")
			flags = append(flags, flag)
		} else {
			args = append(args, arg)
		}
	}
	return flags, args
}