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

	// No prior arg means nothing can require a variable yet.
	if e != nil && e.Taskfile != nil && len(args) > 1 {
		if suggs, dir, ok := completeRequiredVars(e, args[:len(args)-1], fs); ok {
			return suggs, dir
		}
	}

	return completeTaskNames(e, opts)
}

func NeedsTaskfile(args []string, fs *pflag.FlagSet) bool {
	if !parseContext(args).inTaskContext(fs) {
		return false
	}
	// Reading the Taskfile from standard input would hang the shell on a keystroke.
	f := fs.Lookup("taskfile")
	return f == nil || f.Value.String() != "-"
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
	// Not dead defence: flags.WithFlags() clobbers the sorter NewExecutor set.
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
		templated = templated || (!opts.NoDescriptions && strings.Contains(t.Desc, "{{"))
		out = append(out, t)
	}

	if templated {
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
func completeRequiredVars(e *task.Executor, prior []string, fs *pflag.FlagSet) ([]Suggestion, Directive, bool) {
	taskWords, setVars := parsePriorWords(prior, fs)

	out := make([]Suggestion, 0, 8)
	seen := make(map[string]bool, 8)
	for _, w := range taskWords {
		// FindMatchingTasks resolves aliases and wildcards, and unlike GetTask it
		// does not build the fuzzy model to spell-check a word that is not a task.
		if matches, err := e.FindMatchingTasks(&task.Call{Task: w}); err != nil || len(matches) == 0 {
			continue
		}
		compiled, err := e.FastCompiledTask(&task.Call{Task: w})
		if err != nil || compiled == nil || compiled.Requires == nil {
			continue
		}
		for _, v := range compiled.Requires.Vars {
			if v == nil || v.Name == "" || setVars[v.Name] || seen[v.Name] {
				continue
			}
			seen[v.Name] = true
			values := enumValues(v, compiled.Vars)
			if len(values) == 0 {
				out = append(out, Suggestion{Value: v.Name + "="})
				continue
			}
			for _, val := range values {
				out = append(out, Suggestion{Value: v.Name + "=" + val})
			}
		}
	}

	if len(out) == 0 {
		return nil, 0, false
	}
	// KeepOrder preserves the declaration order of the `requires` block.
	return out, DirectiveNoSpace | DirectiveNoFileComp | DirectiveKeepOrder, true
}

func enumValues(v *ast.VarsWithValidation, vars *ast.Vars) []string {
	resolved := refs.ResolveEnum(v, vars)
	if resolved.Enum == nil {
		return nil
	}
	return resolved.Enum.Value
}
