package ast

import (
	"fmt"
	"slices"

	"github.com/go-task/task/v3/errors"
)

// MergeOverride applies a local override: output is replaced, includes are
// added, vars and env merge by key, dotenv files take priority over the main
// ones, tasks are replaced whole.
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
