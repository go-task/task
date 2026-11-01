package ast

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-task/task/v3/errors"
)

// MergeOverride applies a local Taskfile override to tf.
func (tf *Taskfile) MergeOverride(override *Taskfile) error {
	// An older version is fine, so bumping the shared Taskfile doesn't break
	// existing overrides.
	if override.Version.Major() != tf.Version.Major() || override.Version.GreaterThan(tf.Version) {
		return &errors.TaskfileVersionCheckError{
			URI:           override.Location,
			SchemaVersion: override.Version,
			Message:       fmt.Sprintf("must be a v%d version no newer than the main Taskfile (%s)", tf.Version.Major(), tf.Version),
		}
	}
	if override.Output.IsSet() {
		tf.Output = override.Output
	}
	if override.Method != "" {
		tf.Method = override.Method
	}
	if override.Run != "" {
		tf.Run = override.Run
	}
	if override.Interval != 0 {
		tf.Interval = override.Interval
	}
	if override.UseGitignore != nil {
		tf.UseGitignore = override.UseGitignore
	}
	// A present but empty list clears the main one.
	if override.Set != nil {
		tf.Set = override.Set
	}
	if override.Shopt != nil {
		tf.Shopt = override.Shopt
	}
	for namespace, include := range override.Includes.All() {
		if _, ok := tf.Includes.Get(namespace); ok {
			return fmt.Errorf("task: Taskfile override %q can't redefine include %q", filepath.ToSlash(override.Location), namespace)
		}
		tf.Includes.Set(namespace, include)
	}
	if tf.Vars == nil {
		tf.Vars = NewVars()
	}
	if tf.Env == nil {
		tf.Env = NewVars()
	}
	if tf.Tasks == nil {
		tf.Tasks = NewTasks()
	}
	tf.Vars.Merge(override.Vars, nil)
	tf.Env.Merge(override.Env, nil)
	tf.Dotenv = append(slices.Clone(override.Dotenv), tf.Dotenv...)
	for name, task := range override.Tasks.All(nil) {
		if namespace, local, ok := strings.Cut(name, NamespaceSeparator); ok {
			if include, ok := tf.Includes.Get(namespace); ok && !include.Flatten {
				if tf.includeTaskOverrides == nil {
					tf.includeTaskOverrides = map[string]*Tasks{}
				}
				if tf.includeTaskOverrides[namespace] == nil {
					tf.includeTaskOverrides[namespace] = NewTasks()
				}
				// Merging the include adds the namespace back.
				task.Task = local
				tf.includeTaskOverrides[namespace].Set(local, task)
				continue
			}
		}
		tf.Tasks.Set(name, task)
	}
	return nil
}

// includedTasks returns the tasks of the included Taskfile t2 with the
// override's tasks for its namespace applied, so they get the include's
// namespace, directory and variables like its own tasks.
func (t1 *Taskfile) includedTasks(t2 *Taskfile, include *Include) (*Tasks, error) {
	overrides := t1.includeTaskOverrides[include.Namespace]
	if overrides == nil {
		return t2.Tasks, nil
	}
	tasks := NewTasks()
	for name, task := range t2.Tasks.All(nil) {
		tasks.Set(name, task)
	}
	for name, task := range overrides.All(nil) {
		fullName := include.Namespace + NamespaceSeparator + name
		if namespace, _, ok := strings.Cut(name, NamespaceSeparator); ok {
			if _, nested := t2.Includes.Get(namespace); nested {
				return nil, fmt.Errorf("task: Taskfile override %q can't override %q: only tasks of a direct include can be overridden", task.Location.Taskfile, fullName)
			}
		}
		if include.isTaskExcluded(name) {
			return nil, fmt.Errorf("task: Taskfile override %q can't override %q: the include excludes it", task.Location.Taskfile, fullName)
		}
		tasks.Set(name, task)
	}
	return tasks, nil
}
