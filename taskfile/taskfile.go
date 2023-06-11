package taskfile

import (
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
)

var (
	V3 = semver.MustParse("3")
	V2 = semver.MustParse("2")
)

// Taskfile represents a Taskfile.yml
type Taskfile struct {
	Location   string
	Version    *semver.Version
	Expansions int
	Output     Output
	Method     string
	Includes   *IncludedTaskfiles
	Set        []string
	Shopt      []string
	Vars       *Vars
	Env        *Vars
	Tasks      Tasks
	Silent     bool
	Dotenv     []string
	Run        string
	Interval   time.Duration
}

func (tf *Taskfile) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.MappingNode:
		var taskfile struct {
			Version    *semver.Version
			Expansions int
			Output     Output
			Method     string
			Includes   *IncludedTaskfiles
			Set        []string
			Shopt      []string
			Vars       *Vars
			Env        *Vars
			Tasks      Tasks
			Silent     bool
			Dotenv     []string
			Run        string
			Interval   time.Duration
		}
		if err := node.Decode(&taskfile); err != nil {
			return err
		}
		tf.Version = taskfile.Version
		tf.Expansions = taskfile.Expansions
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
		if tf.Expansions <= 0 {
			tf.Expansions = 2
		}
		if tf.Version == nil {
			return errors.New("task: 'version' is required")
		}
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
