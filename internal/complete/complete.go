// Package complete implements the `task __complete` protocol consumed by the
// shell completion wrappers. The protocol mirrors cobra v2 so a future
// migration stays cheap.
package complete

import "os"

// CommandName is the hidden subcommand the shell wrappers invoke to drive
// completion: `task __complete <words...>`.
const CommandName = "__complete"

// IsActive reports whether the process was invoked in completion mode, i.e.
// the first argument is the __complete subcommand.
func IsActive() bool {
	return len(os.Args) >= 2 && os.Args[1] == CommandName
}

// Directive mirrors cobra's ShellCompDirective bitfield. It is emitted on the
// final output line as `:<directive>` and tells the shell wrapper how to treat
// the suggestions (file fallback, trailing space, ordering, …).
type Directive int

const (
	// DirectiveDefault leaves the shell to perform its default file completion.
	DirectiveDefault Directive = 0
	// DirectiveError signals an error; the shell should not offer completion.
	DirectiveError Directive = 1 << 0
	// DirectiveNoSpace prevents the shell from appending a space after the
	// suggestion (e.g. so `VAR=` can be followed by a value).
	DirectiveNoSpace Directive = 1 << 1
	// DirectiveNoFileComp disables the shell's fallback file completion.
	DirectiveNoFileComp Directive = 1 << 2
	// DirectiveFilterFileExt restricts file completion to the emitted extensions.
	DirectiveFilterFileExt Directive = 1 << 3
	// DirectiveFilterDirs restricts completion to directories.
	DirectiveFilterDirs Directive = 1 << 4
	// DirectiveKeepOrder tells the shell to preserve the emitted order instead
	// of sorting alphabetically.
	DirectiveKeepOrder Directive = 1 << 5
)

// Suggestion is a single completion candidate: the Value inserted on the
// command line and an optional human-readable Description.
type Suggestion struct {
	Value       string
	Description string
}

// Options tunes what the engine emits. The zero value shows everything; use
// DefaultOptions for the default and flip fields off from the __complete flags.
type Options struct {
	ShowAliases      bool
	ShowDescriptions bool
}

// DefaultOptions returns the options used when no completion-control flag is
// passed: aliases and descriptions are both shown.
func DefaultOptions() Options {
	return Options{ShowAliases: true, ShowDescriptions: true}
}

// Completion-control flags. Shell wrappers prepend these to the __complete
// invocation to tune the output (e.g. zsh maps its show-aliases / verbose
// zstyles to them). They are consumed by ParseOptions before the remaining
// args are treated as the user's command line.
const (
	FlagNoAliases      = "--no-aliases"
	FlagNoDescriptions = "--no-descriptions"
)

// ParseOptions peels the leading completion-control flags off args and returns
// the resulting Options together with the remaining args (the user's command
// line to complete). Only leading flags are consumed, so a `--no-aliases` typed
// by the user further down the line is left untouched.
func ParseOptions(args []string) (Options, []string) {
	opts := DefaultOptions()
	for len(args) > 0 {
		switch args[0] {
		case FlagNoAliases:
			opts.ShowAliases = false
		case FlagNoDescriptions:
			opts.ShowDescriptions = false
		default:
			return opts, args
		}
		args = args[1:]
	}
	return opts, args
}
