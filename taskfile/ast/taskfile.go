package ast

import (
	"fmt"
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
	t1.Tasks.Merge(t2.Tasks, include)
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
