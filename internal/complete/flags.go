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
	"output":     {"interleaved", "group", "prefixed"},
	"sort":       {"default", "alphanumeric", "none"},
	"completion": completionShells,
}

// A flag absent here falls back to the shell's default file completion.
var flagDirective = map[string]Directive{
	"taskfile":         DirectiveFilterFileExt,
	"dir":              DirectiveFilterDirs,
	"remote-cache-dir": DirectiveFilterDirs,
	"temp-dir":         DirectiveFilterDirs,
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

// valueFlag identifies the value-taking option in a long flag or shorthand
// group. A nonempty prefix means its value is attached to the same word.
func valueFlag(fs *pflag.FlagSet, word string) (*pflag.Flag, string) {
	if fs == nil {
		return nil, ""
	}
	if strings.HasPrefix(word, "--") {
		name, _, attached := strings.Cut(word[2:], "=")
		if f := fs.Lookup(name); f != nil && flagTakesValue(f) {
			if attached {
				return f, "--" + name + "="
			}
			return f, ""
		}
		return nil, ""
	}
	if !strings.HasPrefix(word, "-") {
		return nil, ""
	}
	for i := 1; i < len(word); i++ {
		f := fs.ShorthandLookup(word[i : i+1])
		if f == nil {
			break
		}
		attached := i+1 < len(word)
		if flagTakesValue(f) {
			if !attached {
				return f, ""
			}
			end := i + 1
			if word[end] == '=' {
				end++
			}
			return f, word[:end]
		}
		if attached && word[i+1] == '=' {
			break // The rest is an explicit boolean value, not more flags.
		}
	}
	return nil, ""
}
