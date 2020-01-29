package taskfile

import "errors"

var (
	// ErrCantUnmarshalIncludedTaskFile is returned for invalid var YAML.
	ErrCantUnmarshalIncludedTaskFile = errors.New("task: can't unmarshal included value")
)

// IncludedTaskFile represents information about included tasksfile
type IncludedTaskFile struct {
	Taskfile string
	Dir      string
}

// IncludedTaskFiles represents information about included tasksfiles
type IncludedTaskFiles = map[string]IncludedTaskFile

// UnmarshalYAML implements yaml.Unmarshaler interface
func (it *IncludedTaskFile) UnmarshalYAML(unmarshal func(interface{}) error) error {
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

	return ErrCantUnmarshalIncludedTaskFile
}
