package ast

import (
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	"go.yaml.in/yaml/v4"

	"github.com/go-task/task/v3/errors"
)

// NamespaceSeparator contains the character that separates namespaces
const NamespaceSeparator = ":"

var V3 = semver.MustParse("3")

// ErrIncludedTaskfilesCantHaveDotenvs is returned when a included Taskfile contains dotenvs
var ErrIncludedTaskfilesCantHaveDotenvs = errors.New("task: Included Taskfiles can't have dotenv declarations. Please, move the dotenv declaration to the main Taskfile")

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
	Tasks    *Tasks
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
	if len(t2.Dotenv) > 0 {
		return ErrIncludedTaskfilesCantHaveDotenvs
	}
	if t2.Output.IsSet() {
		t1.Output = t2.Output
	}
	if t1.Includes == nil {
		t1.Includes = NewIncludes()
	}
	if t1.Vars == nil {
		t1.Vars = NewVars()
	}
	if t1.Env == nil {
		t1.Env = NewVars()
	}
	if t1.Tasks == nil {
		t1.Tasks = NewTasks()
	}
	t1.Vars.Merge(t2.Vars, include)
	t1.Env.Merge(t2.Env, include)
	return t1.Tasks.Merge(t2.Tasks, include, t1.Vars)
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
			Tasks    *Tasks
			Silent   bool
			Dotenv   []string
			Run      string
			Interval time.Duration
		}
		if err := node.Decode(&taskfile); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
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
		if tf.Includes == nil {
			tf.Includes = NewIncludes()
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
		return nil
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("taskfile")
}
