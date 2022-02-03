package taskfile

import (
	"fmt"
	"strconv"
)

// Taskfile represents a Taskfile.yml
type Taskfile struct {
	Version    string
	Expansions int
	Output     string
	Method     string
	Includes   *IncludedTaskfiles
	Vars       *Vars
	Env        *Vars
	Tasks      Tasks
	Setup      *[]Cmd
	Silent     bool
	Dotenv     []string
	Run        string
}

// UnmarshalYAML implements yaml.Unmarshaler interface
func (tf *Taskfile) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var taskfile struct {
		Version    string
		Expansions int
		Output     string
		Method     string
		Includes   *IncludedTaskfiles
		Vars       *Vars
		Env        *Vars
		Setup      *[]Cmd
		Tasks      Tasks
		Silent     bool
		Dotenv     []string
		Run        string
	}
	if err := unmarshal(&taskfile); err != nil {
		return err
	}
	tf.Version = taskfile.Version
	tf.Expansions = taskfile.Expansions
	tf.Output = taskfile.Output
	tf.Method = taskfile.Method
	tf.Includes = taskfile.Includes
	tf.Vars = taskfile.Vars
	tf.Env = taskfile.Env
	tf.Setup = taskfile.Setup
	tf.Tasks = taskfile.Tasks
	tf.Silent = taskfile.Silent
	tf.Dotenv = taskfile.Dotenv
	tf.Run = taskfile.Run
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

// ParsedVersion returns the version as a float64
func (tf *Taskfile) ParsedVersion() (float64, error) {
	v, err := strconv.ParseFloat(tf.Version, 64)
	if err != nil {
		return 0, fmt.Errorf(`task: Could not parse taskfile version "%s": %v`, tf.Version, err)
	}
	return v, nil
}
