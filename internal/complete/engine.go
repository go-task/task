package complete

import (
	"strings"

	"github.com/spf13/pflag"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile/ast"
)

// Complete is the single entry point used by `task __complete`. e may be nil
// when the Taskfile failed to load; flag completion still works in that case.
func Complete(e *task.Executor, fs *pflag.FlagSet, args []string) ([]Suggestion, Directive) {
	knownTasks := taskNames(e)
	ctx := parseContext(args, knownTasks, fs)

	if ctx.afterDash {
		return nil, DirectiveDefault
	}

	if ctx.prev != "" {
		if flag := matchFlagName(fs, ctx.prev); flag != nil && flagTakesValue(flag) {
			return completeFlagValue(flag.Name, ctx.toComplete)
		}
	}

	if strings.HasPrefix(ctx.toComplete, "-") {
		if eqIdx := strings.Index(ctx.toComplete, "="); eqIdx != -1 {
			flagWord := ctx.toComplete[:eqIdx]
			partial := ctx.toComplete[eqIdx+1:]
			if f := matchFlagName(fs, flagWord); f != nil && flagTakesValue(f) {
				return completeFlagValue(f.Name, partial)
			}
		}
		return listFlags(fs), DirectiveNoFileComp
	}

	if ctx.taskName != "" && e != nil && e.Taskfile != nil {
		return completeTaskVars(e, ctx.taskName, ctx.toComplete)
	}

	return completeTaskNames(e), DirectiveNoFileComp
}

func taskNames(e *task.Executor) []string {
	if e == nil || e.Taskfile == nil {
		return nil
	}
	var out []string
	for t := range e.Taskfile.Tasks.Values(nil) {
		if t.Internal {
			continue
		}
		out = append(out, strings.TrimSuffix(t.Task, ":"))
		for _, alias := range t.Aliases {
			out = append(out, strings.TrimSuffix(alias, ":"))
		}
	}
	return out
}

func completeTaskNames(e *task.Executor) []Suggestion {
	if e == nil || e.Taskfile == nil {
		return nil
	}
	tasks, err := e.GetTaskList(task.FilterOutInternal)
	if err != nil {
		return nil
	}
	out := make([]Suggestion, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, Suggestion{
			Value:       strings.TrimSuffix(t.Task, ":"),
			Description: t.Desc,
		})
		for _, alias := range t.Aliases {
			out = append(out, Suggestion{
				Value:       strings.TrimSuffix(alias, ":"),
				Description: t.Desc,
			})
		}
	}
	return out
}

func completeFlagValue(flagName, toComplete string) ([]Suggestion, Directive) {
	if dir, ok := flagDirective[flagName]; ok {
		switch dir {
		case DirectiveFilterFileExt:
			suggs := make([]Suggestion, 0, len(taskfileExtensions))
			for _, ext := range taskfileExtensions {
				suggs = append(suggs, Suggestion{Value: ext})
			}
			return suggs, DirectiveFilterFileExt
		case DirectiveFilterDirs:
			return nil, DirectiveFilterDirs
		default:
			return nil, DirectiveDefault
		}
	}

	if values, ok := flagEnums[flagName]; ok {
		out := make([]Suggestion, 0, len(values))
		for _, v := range values {
			out = append(out, Suggestion{Value: v})
		}
		_ = toComplete
		return out, DirectiveNoFileComp
	}

	return nil, DirectiveDefault
}

func completeTaskVars(e *task.Executor, taskName, toComplete string) ([]Suggestion, Directive) {
	compiled, err := e.FastCompiledTask(&task.Call{Task: taskName})
	if err != nil || compiled == nil || compiled.Requires == nil {
		return nil, DirectiveNoFileComp
	}

	cache := &templater.Cache{Vars: compiled.Vars}
	out := make([]Suggestion, 0, 8)
	for _, v := range compiled.Requires.Vars {
		if v == nil || v.Name == "" {
			continue
		}
		values := enumValues(v.Enum, cache)
		if len(values) == 0 {
			out = append(out, Suggestion{Value: v.Name + "="})
			continue
		}
		for _, val := range values {
			out = append(out, Suggestion{Value: v.Name + "=" + val})
		}
	}
	_ = toComplete
	if len(out) == 0 {
		return nil, DirectiveNoFileComp
	}
	return out, DirectiveNoSpace | DirectiveNoFileComp
}

func enumValues(enum *ast.Enum, cache *templater.Cache) []string {
	if enum == nil {
		return nil
	}
	if len(enum.Value) > 0 {
		return enum.Value
	}
	if enum.Ref == "" {
		return nil
	}
	resolved := templater.ResolveRef(enum.Ref, cache)
	if cache.Err() != nil {
		return nil
	}
	arr, ok := resolved.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil
		}
		out = append(out, s)
	}
	return out
}
