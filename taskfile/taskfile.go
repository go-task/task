package taskfile

import (
	"fmt"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Taskfile represents a Taskfile.yml
type Taskfile struct {
	Version    string
	Expansions int
	Output     Output
	Method     string
	Includes   *IncludedTaskfiles
	Set        []string
	Shopts     []string
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
			Version    string
			Expansions int
			Output     Output
			Method     string
			Includes   *IncludedTaskfiles
			Set        []string
			Shopts     []string
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
		tf.Shopts = taskfile.Shopts
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

// ParsedVersion returns the version as a float64
func (tf *Taskfile) ParsedVersion() (float64, error) {
	v, err := strconv.ParseFloat(tf.Version, 64)
	if err != nil {
		return 0, fmt.Errorf(`task: Could not parse taskfile version "%s": %v`, tf.Version, err)
	}
	return v, nil
}
