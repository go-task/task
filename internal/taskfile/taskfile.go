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
	Version         string
	Expansions      int
	Output          string
	Includes        Includes
	IncludeDefaults *Include
	Vars            Vars
	Env             Vars
	Tasks           Tasks
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
		Includes   yaml.MapSlice
		Vars       Vars
		Env        Vars
		Tasks      Tasks
	}
	if err := unmarshal(&taskfile); err != nil {
		return err
	}
	tf.Version = taskfile.Version
	tf.Expansions = taskfile.Expansions
	tf.Output = taskfile.Output
	includes, defaultInclude, err := IncludesFromYaml(taskfile.Includes)
	if err != nil {
		return err
	}
	tf.Includes = includes
	tf.IncludeDefaults = defaultInclude
	tf.Vars = taskfile.Vars
	tf.Env = taskfile.Env
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
	for _, include := range tf.Includes {
		namespace := include.Namespace
		include.Dir = dir
		if tf.IncludeDefaults != nil {
			include.ApplyDefaults(tf.IncludeDefaults)
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
