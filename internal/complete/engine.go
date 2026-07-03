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
func Complete(e *task.Executor, fs *pflag.FlagSet, args []string, opts Options) ([]Suggestion, Directive) {
	ctx := parseContext(args)

	if ctx.afterDash {
		return nil, DirectiveDefault
	}

	if ctx.prev != "" {
		if flag := matchFlagName(fs, ctx.prev); flag != nil && flagTakesValue(flag) {
			return completeFlagValue(flag.Name, "")
		}
	}

	if strings.HasPrefix(ctx.toComplete, "-") {
		if eqIdx := strings.Index(ctx.toComplete, "="); eqIdx != -1 {
			flagWord := ctx.toComplete[:eqIdx]
			if f := matchFlagName(fs, flagWord); f != nil && flagTakesValue(f) {
				// Return full `--flag=value` candidates: shells match/insert
				// against the whole current token, so bare values never match.
				return completeFlagValue(f.Name, flagWord+"=")
			}
		}
		return listFlags(fs), DirectiveNoFileComp
	}

	// Only a task context needs the task list, so it is loaded lazily here.
	if e != nil && e.Taskfile != nil {
		if taskName := detectTaskName(args, taskNames(e), fs); taskName != "" {
			return completeTaskVars(e, taskName)
		}
	}

	return completeTaskNames(e, opts), DirectiveNoFileComp
}

// NeedsTaskfile reports whether completing args requires a loaded Taskfile.
// Flag-name and flag-value completion (and words after `--`) do not, so the
// caller can skip the potentially expensive Taskfile parse for those keystrokes.
func NeedsTaskfile(args []string, fs *pflag.FlagSet) bool {
	ctx := parseContext(args)
	if ctx.afterDash {
		return false
	}
	if ctx.prev != "" {
		if flag := matchFlagName(fs, ctx.prev); flag != nil && flagTakesValue(flag) {
			return false
		}
	}
	return !strings.HasPrefix(ctx.toComplete, "-")
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

func completeTaskNames(e *task.Executor, opts Options) []Suggestion {
	if e == nil || e.Taskfile == nil {
		return nil
	}
	tasks, err := e.GetTaskList(task.FilterOutInternal)
	if err != nil {
		return nil
	}
	desc := func(t *ast.Task) string {
		if !opts.ShowDescriptions {
			return ""
		}
		return t.Desc
	}
	out := make([]Suggestion, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, Suggestion{
			Value:       strings.TrimSuffix(t.Task, ":"),
			Description: desc(t),
		})
		if !opts.ShowAliases {
			continue
		}
		for _, alias := range t.Aliases {
			out = append(out, Suggestion{
				Value:       strings.TrimSuffix(alias, ":"),
				Description: desc(t),
			})
		}
	}
	return out
}

// completeFlagValue completes the value of a value-taking flag. prefix is empty
// for the separate-argument form (`--output <TAB>`) and `<flag>=` for the inline
// form (`--output=<TAB>`), so enum candidates come back as full `--output=value`
// tokens the shell can match against the current word.
func completeFlagValue(flagName, prefix string) ([]Suggestion, Directive) {
	// Absent keys yield the zero value (DirectiveDefault), which falls through
	// to the enum lookup below.
	switch flagDirective[flagName] {
	case DirectiveFilterFileExt:
		suggs := make([]Suggestion, 0, len(taskfileExtensions))
		for _, ext := range taskfileExtensions {
			suggs = append(suggs, Suggestion{Value: ext})
		}
		return suggs, DirectiveFilterFileExt
	case DirectiveFilterDirs:
		return nil, DirectiveFilterDirs
	}

	if values, ok := flagEnums[flagName]; ok {
		out := make([]Suggestion, 0, len(values))
		for _, v := range values {
			out = append(out, Suggestion{Value: prefix + v})
		}
		return out, DirectiveNoFileComp
	}

	return nil, DirectiveDefault
}

func completeTaskVars(e *task.Executor, taskName string) ([]Suggestion, Directive) {
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
	if len(out) == 0 {
		return nil, DirectiveNoFileComp
	}
	// KeepOrder preserves the declaration order of the `requires` block instead
	// of letting the shell sort the variables alphabetically.
	return out, DirectiveNoSpace | DirectiveNoFileComp | DirectiveKeepOrder
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
