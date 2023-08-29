package ast

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	omap "github.com/go-task/task/v3/internal/omap"
)

// Include represents information about included taskfiles
type Include struct {
	Namespace      string
	Taskfile       string
	Dir            string
	Optional       bool
	Internal       bool
	Aliases        []string
	AdvancedImport bool
	Vars           *Vars
	BaseDir        string // The directory from which the including taskfile was loaded; used to resolve relative paths
}

// Includes represents information about included tasksfiles
type Includes struct {
	omap.OrderedMap[string, Include]
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (includes *Includes) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.MappingNode:
		// NOTE(@andreynering): on this style of custom unmarshalling,
		// even number contains the keys, while odd numbers contains
		// the values.
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			var v Include
			if err := valueNode.Decode(&v); err != nil {
				return err
			}
			v.Namespace = keyNode.Value
			includes.Set(keyNode.Value, v)
		}
		return nil
	}

	return fmt.Errorf("yaml: line %d: cannot unmarshal %s into included taskfiles", node.Line, node.ShortTag())
}

// Len returns the length of the map
func (includes *Includes) Len() int {
	if includes == nil {
		return 0
	}
	return includes.OrderedMap.Len()
}

// Wrapper around OrderedMap.Set to ensure we don't get nil pointer errors
func (includes *Includes) Range(f func(k string, v Include) error) error {
	if includes == nil {
		return nil
	}
	return includes.OrderedMap.Range(f)
}

func (include *Include) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	case yaml.ScalarNode:
		var str string
		if err := node.Decode(&str); err != nil {
			return err
		}
		include.Taskfile = str
		return nil

	case yaml.MappingNode:
		var includedTaskfile struct {
			Taskfile string
			Dir      string
			Optional bool
			Internal bool
			Aliases  []string
			Vars     *Vars
		}
		if err := node.Decode(&includedTaskfile); err != nil {
			return err
		}
		include.Taskfile = includedTaskfile.Taskfile
		include.Dir = includedTaskfile.Dir
		include.Optional = includedTaskfile.Optional
		include.Internal = includedTaskfile.Internal
		include.Aliases = includedTaskfile.Aliases
		include.AdvancedImport = true
		include.Vars = includedTaskfile.Vars
		return nil
	}

	return fmt.Errorf("yaml: line %d: cannot unmarshal %s into included taskfile", node.Line, node.ShortTag())
}

// DeepCopy creates a new instance of IncludedTaskfile and copies
// data by value from the source struct.
func (include *Include) DeepCopy() *Include {
	if include == nil {
		return nil
	}
	return &Include{
		Namespace:      include.Namespace,
		Taskfile:       include.Taskfile,
		Dir:            include.Dir,
		Optional:       include.Optional,
		Internal:       include.Internal,
		AdvancedImport: include.AdvancedImport,
		Vars:           include.Vars.DeepCopy(),
		BaseDir:        include.BaseDir,
	}
}

// FullTaskfilePath returns the fully qualified path to the included taskfile
func (include *Include) FullTaskfilePath() (string, error) {
	return include.resolvePath(include.Taskfile)
}

// FullDirPath returns the fully qualified path to the included taskfile's working directory
func (include *Include) FullDirPath() (string, error) {
	return include.resolvePath(include.Dir)
}

func (include *Include) resolvePath(path string) (string, error) {
	// If the file is remote, we don't need to resolve the path
	if strings.Contains(include.Taskfile, "://") {
		return path, nil
	}

	path, err := execext.Expand(path)
	if err != nil {
		return "", err
	}

	if filepathext.IsAbs(path) {
		return path, nil
	}

	result, err := filepath.Abs(filepathext.SmartJoin(include.BaseDir, path))
	if err != nil {
		return "", fmt.Errorf("task: error resolving path %s relative to %s: %w", path, include.BaseDir, err)
	}

	return result, nil
}
