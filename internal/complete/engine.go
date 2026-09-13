package complete

import (
	"strings"

	"github.com/spf13/pflag"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/refs"
	"github.com/go-task/task/v3/internal/slicesext"
	"github.com/go-task/task/v3/internal/sort"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile/ast"
)

// e may be nil when the Taskfile failed to load; flag completion still works.
func Complete(e *task.Executor, fs *pflag.FlagSet, args []string, opts Options) ([]Suggestion, Directive) {
	ctx := parseContext(args, fs)

	if ctx.afterDash {
		return nil, DirectiveDefault
	}

	if flag := ctx.valueFlag; flag != nil {
		return completeFlagValue(flag.Name, "")
	}

	if strings.HasPrefix(ctx.toComplete, "-") {
		if strings.Contains(ctx.toComplete, "=") {
			if f, prefix := valueFlag(fs, ctx.toComplete); f != nil {
				// Shells match against the whole token, so a bare value never would.
				return completeFlagValue(f.Name, prefix)
			}
		}
		return listFlags(fs), DirectiveNoFileComp
	}

	if e == nil || e.Taskfile == nil {
		return nil, DirectiveNoFileComp
	}
	c := compilerWithGlobals(e.Compiler, ctx.vars)
	if suggs := completeRequiredVars(e, c, ctx); len(suggs) > 0 {
		return suggs, DirectiveNoSpace | DirectiveNoFileComp | DirectiveKeepOrder
	}

	return completeTaskNames(e, c, opts), DirectiveNoFileComp
}

func NeedsTaskfile(args []string, fs *pflag.FlagSet) bool {
	if !parseContext(args, fs).inTaskContext() {
		return false
	}
	// Reading the Taskfile from standard input would hang the shell on a keystroke.
	if fs == nil {
		return true
	}
	f := fs.Lookup("taskfile")
	return f == nil || f.Value.String() != "-"
}

func completeTaskNames(e *task.Executor, c *task.Compiler, opts Options) []Suggestion {
	sorter := e.TaskSorter
	if sorter == nil {
		sorter = sort.AlphaNumericWithRootTasksFirst
	}

	out := make([]Suggestion, 0, e.Taskfile.Tasks.Len())
	seen := make(map[string]bool, e.Taskfile.Tasks.Len())
	add := func(name, desc string) {
		value, partial := suggestedName(name)
		// `*-wildcard-*` has no prefix, and `wildcard-*` / `wildcard-*-*` share one.
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		// Without a desc, a truncated pattern says what its prefix stands for.
		if partial && desc == "" && !opts.NoDescriptions {
			desc = name
		}
		out = append(out, Suggestion{Value: value, Description: desc})
	}

	for t := range e.Taskfile.Tasks.Values(sorter) {
		if t.Internal {
			continue
		}
		desc := ""
		if !opts.NoDescriptions {
			desc = taskDescription(c, t)
		}
		add(t.Task, desc)
		if opts.NoAliases {
			continue
		}
		for _, alias := range t.Aliases {
			add(alias, desc)
		}
	}

	return out
}

// Only a templated description needs variables. A failure leaves this task's
// raw description intact without affecting any other task.
func taskDescription(c *task.Compiler, t *ast.Task) string {
	if c == nil || !strings.Contains(t.Desc, "{{") {
		return t.Desc
	}
	vars, err := taskVariables(c, t, t.Task, nil)
	if err != nil {
		return t.Desc
	}
	cache := &templater.Cache{Vars: vars}
	desc := templater.Replace(t.Desc, cache)
	if cache.Err() != nil {
		return t.Desc
	}
	return desc
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
		return suggest("", taskfileExtensions), DirectiveFilterFileExt
	case DirectiveFilterDirs:
		return nil, DirectiveFilterDirs
	}

	if values, ok := flagEnums[flagName]; ok {
		return suggest(prefix, values), DirectiveNoFileComp
	}

	return nil, DirectiveDefault
}

func suggest(prefix string, values []string) []Suggestion {
	return slicesext.Convert(values, func(v string) Suggestion {
		return Suggestion{Value: prefix + v}
	})
}

// CLI variables are global to the invocation, not scoped to a task, so this
// unions the still-unset requirements of every task named on the line. Reporting
// none lets the caller offer task names instead, which is how the line resolves
// itself: fill in what blocks execution, then add another task.
func completeRequiredVars(e *task.Executor, c *task.Compiler, ctx completionContext) []Suggestion {
	if len(ctx.tasks) == 0 {
		return nil
	}
	var out []Suggestion
	seen := make(map[string]bool, 8)
	for _, w := range ctx.tasks {
		matches, err := e.FindMatchingTasks(&task.Call{Task: w})
		if err != nil || len(matches) == 0 {
			continue
		}
		t := matches[0].Task
		if t.Requires == nil {
			continue
		}
		// Static requirements need no compilation. Resolve task variables at
		// most once, and only if an unset requirement has an enum reference.
		var vars *ast.Vars
		resolvedVars := false
		for _, v := range t.Requires.Vars {
			if v == nil || v.Name == "" || seen[v.Name] {
				continue
			}
			if _, set := ctx.vars.Get(v.Name); set {
				continue
			}
			seen[v.Name] = true
			if v.Enum != nil && v.Enum.Ref != "" && len(v.Enum.Value) == 0 && !resolvedVars {
				resolvedVars = true
				if c != nil {
					vars, _ = taskVariables(c, t, w, matches[0].Wildcards)
				}
			}
			values := enumValues(v, vars)
			if len(values) == 0 {
				out = append(out, Suggestion{Value: v.Name + "="})
				continue
			}
			for _, val := range values {
				out = append(out, Suggestion{Value: v.Name + "=" + val})
			}
		}
	}

	return out
}

func taskVariables(c *task.Compiler, t *ast.Task, name string, wildcards []string) (*ast.Vars, error) {
	vars := ast.NewVars()
	vars.Set("MATCH", ast.Var{Value: wildcards})
	return c.FastGetVariables(t, &task.Call{Task: name, Vars: vars})
}

// Execution merges CLI globals into Taskfile vars before resolving templates.
// Use a separate compiler so completion never mutates the loaded Taskfile or
// carries an assignment into a later request.
func compilerWithGlobals(c *task.Compiler, globals *ast.Vars) *task.Compiler {
	if c == nil || globals.Len() == 0 {
		return c
	}
	vars := ast.NewVars()
	vars.Merge(c.TaskfileVars, nil)
	vars.Merge(globals, nil)
	return &task.Compiler{
		Dir:            c.Dir,
		Entrypoint:     c.Entrypoint,
		UserWorkingDir: c.UserWorkingDir,
		TaskfileEnv:    c.TaskfileEnv,
		TaskfileVars:   vars,
		Logger:         c.Logger,
	}
}

func enumValues(v *ast.VarsWithValidation, vars *ast.Vars) []string {
	resolved := refs.ResolveEnum(v, vars)
	if resolved.Enum == nil {
		return nil
	}
	return resolved.Enum.Value
}
