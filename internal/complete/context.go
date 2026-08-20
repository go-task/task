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

// fs is needed to skip the word after a value-taking flag: `task --dir deploy`
// must not read "deploy" as a task name.
func detectTaskName(args []string, knownTasks []string, fs *pflag.FlagSet) string {
	if len(args) <= 1 {
		return ""
	}

	taskName := ""
	skipNext := false
	for _, w := range args[:len(args)-1] {
		if skipNext {
			skipNext = false
			continue
		}
		if w == "--" {
			return taskName
		}
		if strings.HasPrefix(w, "-") {
			if !strings.Contains(w, "=") {
				if f := matchFlagName(fs, w); f != nil && flagTakesValue(f) {
					skipNext = true
				}
			}
			continue
		}
		if strings.Contains(w, "=") {
			continue
		}
		if slices.Contains(knownTasks, w) {
			taskName = w
		}
	}

	return taskName
}
