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

// parseContext infers the cursor position from args alone. It deliberately
// avoids the task list so flag completion never pays to load it; the task word
// is resolved separately by detectTaskName only once a task context is reached.
func parseContext(args []string) completionContext {
	ctx := completionContext{}
	if len(args) == 0 {
		return ctx
	}

	ctx.toComplete = args[len(args)-1]
	if len(args) >= 2 {
		ctx.prev = args[len(args)-2]
	}

	if slices.Contains(args[:len(args)-1], "--") {
		ctx.afterDash = true
	}

	return ctx
}

// detectTaskName scans args for the task word the cursor is completing under
// (e.g. "deploy" in `task deploy ENV=<tab>`). fs is needed to skip the word
// following a value-taking flag, otherwise `task --dir deploy` would mistake
// "deploy" (the directory) for a task name.
func detectTaskName(args []string, knownTasks []string, fs *pflag.FlagSet) string {
	if len(args) <= 1 {
		return ""
	}

	known := make(map[string]struct{}, len(knownTasks))
	for _, t := range knownTasks {
		known[t] = struct{}{}
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
		if _, ok := known[w]; ok {
			taskName = w
		}
	}

	return taskName
}
