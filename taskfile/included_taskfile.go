package taskfile

import (
	"errors"
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// IncludedTaskfile represents information about included taskfiles
type IncludedTaskfile struct {
	Taskfile       string
	Dir            string
	Optional       bool
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
	if node.Kind != yaml.MappingNode {
		return errors.New("task: includes is not a map")
	}

	// NOTE(@andreynering): on this style of custom unmarsheling,
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

// Len returns the length of the map
func (tfs *IncludedTaskfiles) Len() int {
	if tfs == nil {
		return 0
	}
	return len(tfs.Keys)
}

// Merge merges the given IncludedTaskfiles into the caller one
func (tfs *IncludedTaskfiles) Merge(other *IncludedTaskfiles) {
	_ = other.Range(func(key string, value IncludedTaskfile) error {
		tfs.Set(key, value)
		return nil
	})
}

// Set sets a value to a given key
func (tfs *IncludedTaskfiles) Set(key string, includedTaskfile IncludedTaskfile) {
	if tfs.Mapping == nil {
		tfs.Mapping = make(map[string]IncludedTaskfile, 1)
	}
	if !stringSliceContains(tfs.Keys, key) {
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
		Optional bool
		Vars     *Vars
	}
	if err := unmarshal(&includedTaskfile); err != nil {
		return err
	}
	it.Taskfile = includedTaskfile.Taskfile
	it.Dir = includedTaskfile.Dir
	it.Optional = includedTaskfile.Optional
	it.AdvancedImport = true
	it.Vars = includedTaskfile.Vars
	return nil
}

// FullTaskfilePath returns the fully qualified path to the included taskfile
func (it *IncludedTaskfile) FullTaskfilepath() (string, error) {
	return it.resolvePath(it.BaseDir, it.Taskfile)
}

// FullDirPath returns the fully qualified path to the included taskfile's working directory
func (it *IncludedTaskfile) FullDirPath() (string, error) {
	return it.resolvePath(it.BaseDir, it.Dir)
}

func (it *IncludedTaskfile) resolvePath(baseDir string, path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}

	result, err := filepath.Abs(filepath.Join(baseDir, path))
	if err != nil {
		return "", fmt.Errorf("resolving path %s relative to %s: %w", path, baseDir, err)
	}

	return result, nil
}
