package complete

import (
	"slices"
	"strings"

	"github.com/spf13/pflag"
)

type completionContext struct {
	toComplete string
	prev       string
	afterDash  bool
}

// Infers the cursor position from args alone, so flag completion never loads
// the task list.
func parseContext(args []string) completionContext {
	ctx := completionContext{}
	if len(args) == 0 {
		return ctx
	}

	ctx.toComplete = args[len(args)-1]
	if len(args) >= 2 {
		ctx.prev = args[len(args)-2]
	}

	ctx.afterDash = slices.Contains(args[:len(args)-1], "--")

	return ctx
}

func (ctx completionContext) flagValue(fs *pflag.FlagSet) *pflag.Flag {
	if f := matchFlagName(fs, ctx.prev); f != nil && flagTakesValue(f) {
		return f
	}
	return nil
}

func (ctx completionContext) inTaskContext(fs *pflag.FlagSet) bool {
	return !ctx.afterDash && ctx.flagValue(fs) == nil && !strings.HasPrefix(ctx.toComplete, "-")
}

// parsePriorWords splits the words before the cursor into task candidates and
// the names of the variables already set. fs is needed to skip the word after a
// value-taking flag: `task --dir deploy` must not read "deploy" as a task name.
func parsePriorWords(prior []string, fs *pflag.FlagSet) ([]string, map[string]bool) {
	var tasks []string
	setVars := make(map[string]bool, len(prior))

	skipNext := false
	for _, w := range prior {
		if skipNext {
			skipNext = false
			continue
		}
		if strings.HasPrefix(w, "-") {
			if !strings.Contains(w, "=") {
				if f := matchFlagName(fs, w); f != nil && flagTakesValue(f) {
					skipNext = true
				}
			}
			continue
		}
		if name, _, ok := strings.Cut(w, "="); ok {
			setVars[name] = true
			continue
		}
		tasks = append(tasks, w)
	}

	return tasks, setVars
}
