package complete

import (
	"strings"

	"github.com/spf13/pflag"

	"github.com/go-task/task/v3/taskfile/ast"
)

type completionContext struct {
	toComplete string
	valueFlag  *pflag.Flag
	afterDash  bool
	tasks      []string
	vars       *ast.Vars
}

// Walk only the completed words. A pending value consumes the next word even
// when it looks like a flag or "--", just as pflag does.
func parseContext(args []string, fs *pflag.FlagSet) completionContext {
	ctx := completionContext{}
	if len(args) == 0 {
		return ctx
	}
	ctx.toComplete = args[len(args)-1]
	for _, word := range args[:len(args)-1] {
		if ctx.valueFlag != nil {
			ctx.valueFlag = nil
			continue
		}
		if word == "--" {
			ctx.afterDash = true
			break
		}
		if strings.HasPrefix(word, "-") && word != "-" {
			flag, prefix := valueFlag(fs, word)
			if prefix == "" {
				ctx.valueFlag = flag
			}
			continue
		}
		if name, value, ok := strings.Cut(word, "="); ok {
			if ctx.vars == nil {
				ctx.vars = ast.NewVars()
			}
			ctx.vars.Set(name, ast.Var{Value: value})
			continue
		}
		ctx.tasks = append(ctx.tasks, word)
	}
	return ctx
}

func (ctx completionContext) inTaskContext() bool {
	return !ctx.afterDash && ctx.valueFlag == nil && !strings.HasPrefix(ctx.toComplete, "-")
}
