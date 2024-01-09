package ast

import (
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v3"
)

// NamespaceSeparator contains the character that separates namespaces
const NamespaceSeparator = ":"

var V3 = semver.MustParse("3")

// Taskfile is the abstract syntax tree for a Taskfile
type Taskfile struct {
	Location string
	Version  *semver.Version
	Output   Output
	Method   string
	Includes *Includes
	Set      []string
	Shopt    []string
	Vars     *Vars
	Env      *Vars
	Tasks    Tasks
	Silent   bool
	Dotenv   []string
	Run      string
	Interval time.Duration
}

// Merge merges the second Taskfile into the first
func (t1 *Taskfile) Merge(t2 *Taskfile, include *Include) error {
	if !t1.Version.Equal(t2.Version) {
		return fmt.Errorf(`task: Taskfiles versions should match. First is "%s" but second is "%s"`, t1.Version, t2.Version)
	}
	if t2.Output.IsSet() {
		t1.Output = t2.Output
	}

	if t1.Vars == nil {
		t1.Vars = &Vars{}
	}
	if t1.Env == nil {
		t1.Env = &Vars{}
	}
	t1.Vars.Merge(t2.Vars)
	t1.Env.Merge(t2.Env)

	if err := t2.Tasks.Range(func(k string, v *Task) error {
		// We do a deep copy of the task struct here to ensure that no data can
		// be changed elsewhere once the taskfile is merged.
		task := v.DeepCopy()

		// Set the task to internal if EITHER the included task or the included
		// taskfile are marked as internal
		task.Internal = task.Internal || (include != nil && include.Internal)

		// Add namespaces to dependencies, commands and aliases
		for _, dep := range task.Deps {
			if dep != nil && dep.Task != "" {
				dep.Task = taskNameWithNamespace(dep.Task, include.Namespace)
			}
		}
		for _, cmd := range task.Cmds {
			if cmd != nil && cmd.Task != "" {
				cmd.Task = taskNameWithNamespace(cmd.Task, include.Namespace)
			}
		}
		for i, alias := range task.Aliases {
			task.Aliases[i] = taskNameWithNamespace(alias, include.Namespace)
		}
		// Add namespace aliases
		if include != nil {
			for _, namespaceAlias := range include.Aliases {
				task.Aliases = append(task.Aliases, taskNameWithNamespace(task.Task, namespaceAlias))
				for _, alias := range v.Aliases {
					task.Aliases = append(task.Aliases, taskNameWithNamespace(alias, namespaceAlias))
				}
			}
		}

		// Add the task to the merged taskfile
		taskNameWithNamespace := taskNameWithNamespace(k, include.Namespace)
		task.Task = taskNameWithNamespace
		t1.Tasks.Set(taskNameWithNamespace, task)

		return nil
	}); err != nil {
		return err
	}

	// If the included Taskfile has a default task and the parent namespace has
	// no task with a matching name, we can add an alias so that the user can
	// run the included Taskfile's default task without specifying its full
	// name. If the parent namespace has aliases, we add another alias for each
	// of them.
	if t2.Tasks.Get("default") != nil && t1.Tasks.Get(include.Namespace) == nil {
		defaultTaskName := fmt.Sprintf("%s:default", include.Namespace)
		t1.Tasks.Get(defaultTaskName).Aliases = append(t1.Tasks.Get(defaultTaskName).Aliases, include.Namespace)
		t1.Tasks.Get(defaultTaskName).Aliases = append(t1.Tasks.Get(defaultTaskName).Aliases, include.Aliases...)
	}

	return nil
}

func (tf *Taskfile) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.MappingNode:
		var taskfile struct {
			Version  *semver.Version
			Output   Output
			Method   string
			Includes *Includes
			Set      []string
			Shopt    []string
			Vars     *Vars
			Env      *Vars
			Tasks    Tasks
			Silent   bool
			Dotenv   []string
			Run      string
			Interval time.Duration
		}
		if err := node.Decode(&taskfile); err != nil {
			return err
		}
		tf.Version = taskfile.Version
		tf.Output = taskfile.Output
		tf.Method = taskfile.Method
		tf.Includes = taskfile.Includes
		tf.Set = taskfile.Set
		tf.Shopt = taskfile.Shopt
		tf.Vars = taskfile.Vars
		tf.Env = taskfile.Env
		tf.Tasks = taskfile.Tasks
		tf.Silent = taskfile.Silent
		tf.Dotenv = taskfile.Dotenv
		tf.Run = taskfile.Run
		tf.Interval = taskfile.Interval
		if tf.Vars == nil {
			tf.Vars = &Vars{}
		}
		if tf.Env == nil {
			tf.Env = &Vars{}
		}
		return nil
	}

	return fmt.Errorf("yaml: line %d: cannot unmarshal %s into taskfile", node.Line, node.ShortTag())
}

func taskNameWithNamespace(taskName string, namespace string) string {
	if strings.HasPrefix(taskName, NamespaceSeparator) {
		return strings.TrimPrefix(taskName, NamespaceSeparator)
	}
	return fmt.Sprintf("%s%s%s", namespace, NamespaceSeparator, taskName)
}
