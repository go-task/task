package complete

import (
	"sort"
	"strings"

	"github.com/spf13/pflag"
)

// flagEnums lists allowed values for enum-style flags. Keep in sync with the
// help strings in internal/flags/flags.go.
var flagEnums = map[string][]string{
	"output":     {"interleaved", "group", "prefixed"},
	"sort":       {"default", "alphanumeric", "none"},
	"completion": {"bash", "zsh", "fish", "powershell"},
}

// flagDirective maps value-taking flags to a file-completion directive.
// DirectiveDefault entries (and any flag absent here) fall back to the shell's
// default file completion.
var flagDirective = map[string]Directive{
	"taskfile":         DirectiveFilterFileExt,
	"dir":              DirectiveFilterDirs,
	"remote-cache-dir": DirectiveFilterDirs,
	"cacert":           DirectiveDefault,
	"cert":             DirectiveDefault,
	"cert-key":         DirectiveDefault,
}

var taskfileExtensions = []string{"yml", "yaml"}

// flagTakesValue is false for boolean switches (NoOptDefVal == "true").
func flagTakesValue(f *pflag.Flag) bool {
	return f.NoOptDefVal == ""
}

// listFlags walks fs at call time so experiment-gated flags appear or
// disappear based on the active experiments.
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
	sort.Slice(out, func(i, j int) bool { return out[i].Value < out[j].Value })
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
