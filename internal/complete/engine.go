package complete

import (
	"strings"

	"github.com/spf13/pflag"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/refs"
	"github.com/go-task/task/v3/internal/slicesext"
	"github.com/go-task/task/v3/internal/sort"
	"github.com/go-task/task/v3/taskfile/ast"
)

// e may be nil when the Taskfile failed to load; flag completion still works.
func Complete(e *task.Executor, fs *pflag.FlagSet, args []string, opts Options) ([]Suggestion, Directive) {
	ctx := parseContext(args)

	if ctx.afterDash {
		return nil, DirectiveDefault
	}

	if flag := ctx.flagValue(fs); flag != nil {
		return completeFlagValue(flag.Name, "")
	}

	if strings.HasPrefix(ctx.toComplete, "-") {
		if flagWord, _, ok := strings.Cut(ctx.toComplete, "="); ok {
			if f := matchFlagName(fs, flagWord); f != nil && flagTakesValue(f) {
				// Shells match against the whole token, so a bare value never would.
				return completeFlagValue(f.Name, flagWord+"=")
			}
		}
		return listFlags(fs), DirectiveNoFileComp
	}

	// No prior arg means no task word, so `task <tab>` never builds the list.
	if e != nil && e.Taskfile != nil && len(args) > 1 {
		if taskName := detectTaskName(args, taskNames(e), fs); taskName != "" {
			return completeTaskVars(e, taskName)
		}
	}

	return completeTaskNames(e, opts)
}

func NeedsTaskfile(args []string, fs *pflag.FlagSet) bool {
	return parseContext(args).inTaskContext(fs)
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
		name, _ := suggestedName(t.Task)
		out = append(out, name)
		for _, alias := range t.Aliases {
			name, _ := suggestedName(alias)
			out = append(out, name)
		}
	}
	return out
}

func completeTaskNames(e *task.Executor, opts Options) ([]Suggestion, Directive) {
	if e == nil || e.Taskfile == nil {
		return nil, DirectiveNoFileComp
	}
	tasks := listTasks(e, opts)
	desc := func(t *ast.Task) string {
		if opts.NoDescriptions {
			return ""
		}
		return t.Desc
	}

	out := make([]Suggestion, 0, len(tasks))
	seen := make(map[string]bool, len(tasks))
	anyPartial := false
	add := func(name, desc string) {
		value, partial := suggestedName(name)
		// `*-wildcard-*` has no prefix, and `wildcard-*` / `wildcard-*-*` share one.
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		if partial {
			anyPartial = true
			if desc == "" && !opts.NoDescriptions {
				desc = name
			}
		}
		out = append(out, Suggestion{Value: value, Description: desc})
	}

	for _, t := range tasks {
		add(t.Task, desc(t))
		if opts.NoAliases {
			continue
		}
		for _, alias := range t.Aliases {
			add(alias, desc(t))
		}
	}

	// A truncated pattern is half a name: the cursor must stay against it.
	if anyPartial {
		return out, DirectiveNoSpace | DirectiveNoFileComp
	}
	return out, DirectiveNoFileComp
}

// GetTaskList compiles every task, on every keystroke, and a description is the
// only compiled field read: worth its cost only when one holds a template.
func listTasks(e *task.Executor, opts Options) []*ast.Task {
	sorter := e.TaskSorter
	if sorter == nil {
		sorter = sort.AlphaNumericWithRootTasksFirst
	}

	out := make([]*ast.Task, 0, e.Taskfile.Tasks.Len())
	templated := false
	for t := range e.Taskfile.Tasks.Values(sorter) {
		if t.Internal {
			continue
		}
		templated = templated || strings.Contains(t.Desc, "{{")
		out = append(out, t)
	}

	if !opts.NoDescriptions && templated {
		// The uncompiled tasks keep one broken task from emptying the list.
		if compiled, err := e.GetTaskList(task.FilterOutInternal); err == nil {
			return compiled
		}
	}
	return out
}

// A pattern is truncated at its `*`: it is not runnable, `.MATCH` would be empty.
func suggestedName(name string) (string, bool) {
	if prefix, _, ok := strings.Cut(name, "*"); ok {
		return prefix, true
	}
	return strings.TrimRight(name, ":"), false
}

// prefix is `<flag>=` for the inline form, so a candidate matches the whole token.
func completeFlagValue(flagName, prefix string) ([]Suggestion, Directive) {
	// An absent key yields DirectiveDefault, falling through to the enums.
	switch flagDirective[flagName] {
	case DirectiveFilterFileExt:
		exts := slicesext.Convert(taskfileExtensions, func(ext string) Suggestion {
			return Suggestion{Value: ext}
		})
		return exts, DirectiveFilterFileExt
	case DirectiveFilterDirs:
		return nil, DirectiveFilterDirs
	}

	if values, ok := flagEnums[flagName]; ok {
		out := slicesext.Convert(values, func(v string) Suggestion {
			return Suggestion{Value: prefix + v}
		})
		return out, DirectiveNoFileComp
	}

	return nil, DirectiveDefault
}

func completeTaskVars(e *task.Executor, taskName string) ([]Suggestion, Directive) {
	compiled, err := e.FastCompiledTask(&task.Call{Task: taskName})
	if err != nil || compiled == nil || compiled.Requires == nil {
		return nil, DirectiveNoFileComp
	}

	out := make([]Suggestion, 0, 8)
	for _, v := range compiled.Requires.Vars {
		if v == nil || v.Name == "" {
			continue
		}
		values := enumValues(v, compiled.Vars)
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
	// KeepOrder preserves the declaration order of the `requires` block.
	return out, DirectiveNoSpace | DirectiveNoFileComp | DirectiveKeepOrder
}

func enumValues(v *ast.VarsWithValidation, vars *ast.Vars) []string {
	resolved := refs.ResolveEnum(v, vars)
	if resolved.Enum == nil {
		return nil
	}
	return resolved.Enum.Value
}
