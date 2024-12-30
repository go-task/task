package ast

import (
	"github.com/elliotchance/orderedmap/v2"
	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/deepcopy"
)

// Include represents information about included taskfiles
type Include struct {
	Namespace      string
	Taskfile       string
	Dir            string
	Optional       bool
	Internal       bool
	Aliases        []string
	Excludes       []string
	AdvancedImport bool
	Vars           *Vars
	Flatten        bool
}

// Includes represents information about included taskfiles
type Includes struct {
	om *orderedmap.OrderedMap[string, *Include]
}

type IncludeElement orderedmap.Element[string, *Include]

func NewIncludes(els ...*IncludeElement) *Includes {
	includes := &Includes{
		om: orderedmap.NewOrderedMap[string, *Include](),
	}
	for _, el := range els {
		includes.Set(el.Key, el.Value)
	}
	return includes
}

func (includes *Includes) Len() int {
	if includes == nil || includes.om == nil {
		return 0
	}
	return includes.om.Len()
}

func (includes *Includes) Get(key string) (*Include, bool) {
	if includes == nil || includes.om == nil {
		return &Include{}, false
	}
	return includes.om.Get(key)
}

func (includes *Includes) Set(key string, value *Include) bool {
	if includes == nil {
		includes = NewIncludes()
	}
	if includes.om == nil {
		includes.om = orderedmap.NewOrderedMap[string, *Include]()
	}
	return includes.om.Set(key, value)
}

func (includes *Includes) Range(f func(k string, v *Include) error) error {
	if includes == nil || includes.om == nil {
		return nil
	}
	for pair := includes.om.Front(); pair != nil; pair = pair.Next() {
		if err := f(pair.Key, pair.Value); err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (includes *Includes) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.MappingNode:
		// NOTE: orderedmap does not have an unmarshaler, so we have to decode
		// the map manually. We increment over 2 values at a time and assign
		// them as a key-value pair.
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			// Decode the value node into an Include struct
			var v Include
			if err := valueNode.Decode(&v); err != nil {
				return errors.NewTaskfileDecodeError(err, node)
			}

			// Set the include namespace
			v.Namespace = keyNode.Value

			// Add the include to the ordered map
			includes.Set(keyNode.Value, &v)
		}
		return nil
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("includes")
}

func (include *Include) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	case yaml.ScalarNode:
		var str string
		if err := node.Decode(&str); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		include.Taskfile = str
		return nil

	case yaml.MappingNode:
		var includedTaskfile struct {
			Taskfile string
			Dir      string
			Optional bool
			Internal bool
			Flatten  bool
			Aliases  []string
			Excludes []string
			Vars     *Vars
		}
		if err := node.Decode(&includedTaskfile); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		include.Taskfile = includedTaskfile.Taskfile
		include.Dir = includedTaskfile.Dir
		include.Optional = includedTaskfile.Optional
		include.Internal = includedTaskfile.Internal
		include.Aliases = includedTaskfile.Aliases
		include.Excludes = includedTaskfile.Excludes
		include.AdvancedImport = true
		include.Vars = includedTaskfile.Vars
		include.Flatten = includedTaskfile.Flatten
		return nil
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("include")
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
		Excludes:       deepcopy.Slice(include.Excludes),
		AdvancedImport: include.AdvancedImport,
		Vars:           include.Vars.DeepCopy(),
		Flatten:        include.Flatten,
	}
}
