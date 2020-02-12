package taskfile

import "errors"

var (
	// ErrCantUnmarshalIncludedTaskfile is returned for invalid var YAML.
	ErrCantUnmarshalIncludedTaskfile = errors.New("task: can't unmarshal included value")
)

// IncludedTaskfile represents information about included tasksfile
type IncludedTaskfile struct {
	Taskfile string
	Dir      string
}

// IncludedTaskfiles represents information about included tasksfiles
type IncludedTaskfiles = map[string]IncludedTaskfile

// UnmarshalYAML implements yaml.Unmarshaler interface
func (it *IncludedTaskfile) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err == nil {
		it.Taskfile = str
		return nil
	}

	var includedTaskfile struct {
		Taskfile string
		Dir      string
	}
	if err := unmarshal(&includedTaskfile); err == nil {
		it.Dir = includedTaskfile.Dir
		it.Taskfile = includedTaskfile.Taskfile
		return nil
	}

	return ErrCantUnmarshalIncludedTaskfile
}
