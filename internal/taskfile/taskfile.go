package taskfile

import (
	"os"

	yaml "gopkg.in/yaml.v2"
)

// Taskfile represents a Taskfile.yml
type Taskfile struct {
	Version    string
	Expansions int
	Output     string
	Includes   map[string]*Includes
	Vars       Vars
	Tasks      Tasks
}

// UnmarshalYAML implements yaml.Unmarshaler interface
func (tf *Taskfile) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal(&tf.Tasks); err == nil {
		tf.Version = "1"
		return nil
	}

	var taskfile struct {
		Version    string
		Expansions int
		Output     string
		Includes   map[string]*Includes
		Vars       Vars
		Tasks      Tasks
	}
	if err := unmarshal(&taskfile); err != nil {
		return err
	}
	tf.Version = taskfile.Version
	tf.Expansions = taskfile.Expansions
	tf.Output = taskfile.Output
	tf.Includes = taskfile.Includes
	tf.Vars = taskfile.Vars
	tf.Tasks = taskfile.Tasks
	if tf.Expansions <= 0 {
		tf.Expansions = 2
	}
	return nil
}

// LoadFromPath parses a local Taskfile
func LoadFromPath(path string) (*Taskfile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	var t Taskfile
	return &t, yaml.NewDecoder(f).Decode(&t)
}
