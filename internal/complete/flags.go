package complete

import (
	"slices"
	"strings"

	"github.com/spf13/pflag"
)

// TestCompletionShells keeps this in step with the scripts the root package serves.
var completionShells = []string{"bash", "zsh", "fish", "powershell", "nu"}

// Keep in sync with the help strings in internal/flags/flags.go.
var flagEnums = map[string][]string{
	"output":         {"interleaved", "group", "prefixed"},
	"sort":           {"default", "alphanumeric", "none"},
	"completion":     completionShells,
	"new-completion": completionShells,
}

// A flag absent here falls back to the shell's default file completion.
var flagDirective = map[string]Directive{
	"taskfile":         DirectiveFilterFileExt,
	"dir":              DirectiveFilterDirs,
	"remote-cache-dir": DirectiveFilterDirs,
}

var taskfileExtensions = []string{"yml", "yaml"}

func flagTakesValue(f *pflag.Flag) bool {
	return f.NoOptDefVal == ""
}

// Walks fs at call time so experiment-gated flags follow the active experiments.
func listFlags(fs *pflag.FlagSet) []Suggestion {
	if fs == nil {
		return nil
	}
	out := make([]Suggestion, 0, 64)
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Hidden || f.Deprecated != "" {
			return
		}
		out = append(out, Suggestion{
			Value:       "--" + f.Name,
			Description: f.Usage,
		})
		if f.Shorthand != "" {
			out = append(out, Suggestion{
				Value:       "-" + f.Shorthand,
				Description: f.Usage,
			})
		}
	})
	slices.SortFunc(out, func(a, b Suggestion) int { return strings.Compare(a.Value, b.Value) })
	return out
}

func matchFlagName(fs *pflag.FlagSet, word string) *pflag.Flag {
	if fs == nil {
		return nil
	}
	switch {
	case strings.HasPrefix(word, "--"):
		return fs.Lookup(strings.TrimPrefix(word, "--"))
	case strings.HasPrefix(word, "-") && len(word) == 2:
		return fs.ShorthandLookup(word[1:])
	}
	return nil
}
