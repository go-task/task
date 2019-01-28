package taskfile

import (
	"errors"

	"github.com/spf13/afero"
	yaml "gopkg.in/yaml.v2"
)

// ErrIncludedTaskfilesCantHaveIncludes is returned when a included Taskfile contains includes
var (
	ErrIncludedTaskfilesCantHaveIncludes = errors.New(
		`task: Included Taskfiles can't have includes. 
				Please, move the include to the main Taskfile`,
	)

	// AppFS provides a filesystem passthrough, very useful for testing environments
	AppFS = afero.NewOsFs()
)

// Taskfile represents a Taskfile.yml
type Taskfile struct {
	Version    string
	Expansions int
	Output     string
	Includes   Includes
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
		Includes   map[string]*Include
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
	f, err := AppFS.Open(path)
	if err != nil {
		return nil, err
	}
	var t Taskfile
	return &t, yaml.NewDecoder(f).Decode(&t)
}

func (tf *Taskfile) ProcessIncludes(dir string) error {
	defaults, defaults_available := tf.Includes[".defaults"]

	for namespace, include := range tf.Includes {
		if namespace == ".defaults" {
			continue
		}
		include.Dir = dir
		if defaults_available {
			include.ApplyDefaults(defaults)
		}
		include.ApplySettingsByNamespace(namespace)
		includedTaskfile, err := include.LoadTaskfile()

		if err != nil {
			return err
		}
		if len(includedTaskfile.Includes) > 0 {
			return ErrIncludedTaskfilesCantHaveIncludes
		}
		if err = Merge(include, tf, includedTaskfile, namespace); err != nil {
			return err
		}
	}
	return nil
}
