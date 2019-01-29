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

type taskfilePipeline struct {
	inc   *Include
	tf    *Taskfile
	order int
	err   error
}

func (tf *Taskfile) ProcessIncludes(dir string) error {
	ch := make(chan taskfilePipeline, len(tf.Includes))
	defer close(ch)
	for i, include := range tf.Includes {
		namespace := include.Namespace
		include.Dir = dir
		if tf.IncludeDefaults != nil {
			include.ApplyDefaults(tf.IncludeDefaults)
		}
		include.ApplySettingsByNamespace(namespace)
		go func(pipeline chan taskfilePipeline, include *Include, i int) {
			includedTaskfile, err := include.LoadTaskfile()
			pipeline <- taskfilePipeline{include, includedTaskfile, i, err}
		}(ch, include, i)
	}

	list := make([]taskfilePipeline, len(tf.Includes))

	for range tf.Includes {
		tp := <-ch
		list[tp.order] = tp
	}

	for _, tp := range list {
		if tp.err != nil {
			return tp.err
		}
		if len(tp.tf.Includes) > 0 {
			return ErrIncludedTaskfilesCantHaveIncludes
		}
		if err := Merge(tp.inc, tf, tp.tf, tp.inc.Namespace); err != nil {
			return err
		}
	}
	return nil
}
