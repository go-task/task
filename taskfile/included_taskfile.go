package taskfile

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

// IncludedTaskfile represents information about included taskfiles
type IncludedTaskfile struct {
	Taskfile       string
	Dir            string
	Optional       bool
	Internal       bool
	Aliases        []string
	AdvancedImport bool
	Vars           *Vars
	BaseDir        string // The directory from which the including taskfile was loaded; used to resolve relative paths
}

// IncludedTaskfiles represents information about included tasksfiles
type IncludedTaskfiles struct {
	Keys    []string
	Mapping map[string]IncludedTaskfile
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (tfs *IncludedTaskfiles) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.MappingNode:
		// NOTE(@andreynering): on this style of custom unmarshalling,
		// even number contains the keys, while odd numbers contains
		// the values.
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			var v IncludedTaskfile
			if err := valueNode.Decode(&v); err != nil {
				return err
			}
			tfs.Set(keyNode.Value, v)
		}
		return nil
	}

	return fmt.Errorf("yaml: line %d: cannot unmarshal %s into included taskfiles", node.Line, node.ShortTag())
}

// Len returns the length of the map
func (tfs *IncludedTaskfiles) Len() int {
	if tfs == nil {
		return 0
	}
	return len(tfs.Keys)
}

// Set sets a value to a given key
func (tfs *IncludedTaskfiles) Set(key string, includedTaskfile IncludedTaskfile) {
	if tfs.Mapping == nil {
		tfs.Mapping = make(map[string]IncludedTaskfile, 1)
	}
	if !slices.Contains(tfs.Keys, key) {
		tfs.Keys = append(tfs.Keys, key)
	}
	tfs.Mapping[key] = includedTaskfile
}

// Range allows you to loop into the included taskfiles in its right order
func (tfs *IncludedTaskfiles) Range(yield func(key string, includedTaskfile IncludedTaskfile) error) error {
	if tfs == nil {
		return nil
	}
	for _, k := range tfs.Keys {
		if err := yield(k, tfs.Mapping[k]); err != nil {
			return err
		}
	}
	return nil
}

func (it *IncludedTaskfile) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	case yaml.ScalarNode:
		var str string
		if err := node.Decode(&str); err != nil {
			return err
		}
		it.Taskfile = str
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
		it.Taskfile = includedTaskfile.Taskfile
		it.Dir = includedTaskfile.Dir
		it.Optional = includedTaskfile.Optional
		it.Internal = includedTaskfile.Internal
		it.Aliases = includedTaskfile.Aliases
		it.AdvancedImport = true
		it.Vars = includedTaskfile.Vars
		return nil
	}

	return fmt.Errorf("yaml: line %d: cannot unmarshal %s into included taskfile", node.Line, node.ShortTag())
}

// DeepCopy creates a new instance of IncludedTaskfile and copies
// data by value from the source struct.
func (it *IncludedTaskfile) DeepCopy() *IncludedTaskfile {
	if it == nil {
		return nil
	}
	return &IncludedTaskfile{
		Taskfile:       it.Taskfile,
		Dir:            it.Dir,
		Optional:       it.Optional,
		Internal:       it.Internal,
		AdvancedImport: it.AdvancedImport,
		Vars:           it.Vars.DeepCopy(),
		BaseDir:        it.BaseDir,
	}
}

// FullTaskfilePath returns the fully qualified path to the included taskfile
func (it *IncludedTaskfile) FullTaskfilePath() (string, error) {
	return it.resolvePath(it.Taskfile)
}

// FullDirPath returns the fully qualified path to the included taskfile's working directory
func (it *IncludedTaskfile) FullDirPath() (string, error) {
	return it.resolvePath(it.Dir)
}

func (it *IncludedTaskfile) resolvePath(path string) (string, error) {
	// If the file is remote, we don't need to resolve the path
	if strings.Contains(it.Taskfile, "://") {
		return path, nil
	}

	path, err := execext.Expand(path)
	if err != nil {
		return "", err
	}

	if filepathext.IsAbs(path) {
		return path, nil
	}

	result, err := filepath.Abs(filepathext.SmartJoin(it.BaseDir, path))
	if err != nil {
		return "", fmt.Errorf("task: error resolving path %s relative to %s: %w", path, it.BaseDir, err)
	}

	return result, nil
}
