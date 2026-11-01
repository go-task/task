package ast

import (
	"fmt"
	"slices"

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
			return fmt.Errorf("task: Taskfile override %q can't redefine include %q", override.Location, namespace)
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
		tf.Tasks.Set(name, task)
	}
	return nil
}
