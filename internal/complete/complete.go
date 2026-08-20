// Package complete implements the `task __complete` protocol consumed by the
// shell wrappers. It mirrors cobra v2 so a future migration stays cheap.
package complete

import "os"

const CommandName = "__complete"

func IsActive() bool {
	return len(os.Args) >= 2 && os.Args[1] == CommandName
}

// Directive mirrors cobra's ShellCompDirective bitfield, emitted as `:<n>`.
type Directive int

const (
	DirectiveDefault Directive = 0
	// Never emitted: a failed Taskfile load still leaves flags worth completing.
	DirectiveError         Directive = 1 << 0
	DirectiveNoSpace       Directive = 1 << 1
	DirectiveNoFileComp    Directive = 1 << 2
	DirectiveFilterFileExt Directive = 1 << 3
	DirectiveFilterDirs    Directive = 1 << 4
	DirectiveKeepOrder     Directive = 1 << 5
)

type Suggestion struct {
	Value       string
	Description string
}

// Named after the control flags, so the zero value is the standard set.
type Options struct {
	NoAliases      bool
	NoDescriptions bool
}

// Control flags the shell wrappers prepend to the __complete invocation.
const (
	FlagNoAliases      = "--no-aliases"
	FlagNoDescriptions = "--no-descriptions"
)

// Only leading flags are consumed; a `--no-aliases` typed later is left alone.
func ParseOptions(args []string) (Options, []string) {
	var opts Options
	for len(args) > 0 {
		switch args[0] {
		case FlagNoAliases:
			opts.NoAliases = true
		case FlagNoDescriptions:
			opts.NoDescriptions = true
		default:
			return opts, args
		}
		args = args[1:]
	}
	return opts, args
}
