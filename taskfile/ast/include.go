package ast

import (
	"iter"
	"sync"

	"github.com/elliotchance/orderedmap/v3"
	"go.yaml.in/yaml/v4"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/deepcopy"
)

type (
	// Include represents information about included taskfiles
	Include struct {
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
		Checksum       string
	}
	// Includes is an ordered map of namespaces to includes.
	Includes struct {
		om    *orderedmap.OrderedMap[string, *Include]
		mutex sync.RWMutex
	}
	// An IncludeElement is a key-value pair that is used for initializing an
	// Includes structure.
	IncludeElement orderedmap.Element[string, *Include]
)

// NewIncludes creates a new instance of Includes and initializes it with the
// provided set of elements, if any. The elements are added in the order they
// are passed.
func NewIncludes(els ...*IncludeElement) *Includes {
	includes := &Includes{
		om: orderedmap.NewOrderedMap[string, *Include](),
	}
	for _, el := range els {
		includes.Set(el.Key, el.Value)
	}
	return includes
}

// Len returns the number of includes in the Includes map.
func (includes *Includes) Len() int {
	if includes == nil || includes.om == nil {
		return 0
	}
	defer includes.mutex.RUnlock()
	includes.mutex.RLock()
	return includes.om.Len()
}

// Get returns the value the the include with the provided key and a boolean
// that indicates if the value was found or not. If the value is not found, the
// returned include is a zero value and the bool is false.
func (includes *Includes) Get(key string) (*Include, bool) {
	if includes == nil || includes.om == nil {
		return &Include{}, false
	}
	defer includes.mutex.RUnlock()
	includes.mutex.RLock()
	return includes.om.Get(key)
}

// Set sets the value of the include with the provided key to the provided
// value. If the include already exists, its value is updated. If the include
// does not exist, it is created.
func (includes *Includes) Set(key string, value *Include) bool {
	if includes == nil {
		includes = NewIncludes()
	}
	if includes.om == nil {
		includes.om = orderedmap.NewOrderedMap[string, *Include]()
	}
	defer includes.mutex.Unlock()
	includes.mutex.Lock()
	return includes.om.Set(key, value)
}

// All returns an iterator that loops over all task key-value pairs.
// Range calls the provided function for each include in the map. The function
// receives the include's key and value as arguments. If the function returns
// an error, the iteration stops and the error is returned.
func (includes *Includes) All() iter.Seq2[string, *Include] {
	if includes == nil || includes.om == nil {
		return func(yield func(string, *Include) bool) {}
	}
	return includes.om.AllFromFront()
}

// Keys returns an iterator that loops over all task keys.
func (includes *Includes) Keys() iter.Seq[string] {
	if includes == nil || includes.om == nil {
		return func(yield func(string) bool) {}
	}
	return includes.om.Keys()
}

// Values returns an iterator that loops over all task values.
func (includes *Includes) Values() iter.Seq[*Include] {
	if includes == nil || includes.om == nil {
		return func(yield func(*Include) bool) {}
	}
	return includes.om.Values()
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (includes *Includes) UnmarshalYAML(node *yaml.Node) error {
	if includes == nil || includes.om == nil {
		*includes = *NewIncludes()
	}
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
			Checksum string
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
		include.Checksum = includedTaskfile.Checksum
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
		Aliases:        deepcopy.Slice(include.Aliases),
		Checksum:       include.Checksum,
	}
}
