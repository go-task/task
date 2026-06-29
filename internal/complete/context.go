package complete

import (
	"strings"

	"github.com/spf13/pflag"
)

type completionContext struct {
	toComplete string
	prev       string
	taskName   string
	afterDash  bool
}

// parseContext infers the cursor position from args. fs is needed to skip the
// word following a value-taking flag, otherwise `task --dir deploy` would
// mistake "deploy" (the directory) for a task name.
func parseContext(args []string, knownTasks []string, fs *pflag.FlagSet) completionContext {
	ctx := completionContext{}
	if len(args) == 0 {
		return ctx
	}

	ctx.toComplete = args[len(args)-1]
	if len(args) >= 2 {
		ctx.prev = args[len(args)-2]
	}

	known := make(map[string]struct{}, len(knownTasks))
	for _, t := range knownTasks {
		known[t] = struct{}{}
	}

	skipNext := false
	for _, w := range args[:len(args)-1] {
		if skipNext {
			skipNext = false
			continue
		}
		if w == "--" {
			ctx.afterDash = true
			continue
		}
		if ctx.afterDash {
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
		if strings.Contains(w, "=") {
			continue
		}
		if _, ok := known[w]; ok {
			ctx.taskName = w
		}
	}

	return ctx
}
