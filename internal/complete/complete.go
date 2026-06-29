// Package complete implements the `task __complete` protocol consumed by the
// shell completion wrappers. The protocol mirrors cobra v2 so a future
// migration stays cheap.
package complete

import "os"

const CommandName = "__complete"

func IsActive() bool {
	return len(os.Args) >= 2 && os.Args[1] == CommandName
}

// Directive mirrors cobra's ShellCompDirective bitfield.
type Directive int

const (
	DirectiveDefault       Directive = 0
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
