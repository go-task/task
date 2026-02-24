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
	// Override represents information about overridden taskfiles
	Override struct {
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
	// Overrides is an ordered map of namespaces to overrides.
	Overrides struct {
		om    *orderedmap.OrderedMap[string, *Override]
		mutex sync.RWMutex
	}
	// An OverrideElement is a key-value pair that is used for initializing an
	// Overrides structure.
	OverrideElement orderedmap.Element[string, *Override]
)

// NewOverrides creates a new instance of Overrides and initializes it with the
// provided set of elements, if any. The elements are added in the order they
// are passed.
func NewOverrides(els ...*OverrideElement) *Overrides {
	overrides := &Overrides{
		om: orderedmap.NewOrderedMap[string, *Override](),
	}
	for _, el := range els {
		overrides.Set(el.Key, el.Value)
	}
	return overrides
}

// Len returns the number of overrides in the Overrides map.
func (overrides *Overrides) Len() int {
	if overrides == nil || overrides.om == nil {
		return 0
	}
	defer overrides.mutex.RUnlock()
	overrides.mutex.RLock()
	return overrides.om.Len()
}

// Get returns the value the the override with the provided key and a boolean
// that indicates if the value was found or not. If the value is not found, the
// returned override is a zero value and the bool is false.
func (overrides *Overrides) Get(key string) (*Override, bool) {
	if overrides == nil || overrides.om == nil {
		return &Override{}, false
	}
	defer overrides.mutex.RUnlock()
	overrides.mutex.RLock()
	return overrides.om.Get(key)
}

// Set sets the value of the override with the provided key to the provided
// value. If the override already exists, its value is updated. If the override
// does not exist, it is created.
func (overrides *Overrides) Set(key string, value *Override) bool {
	if overrides == nil {
		overrides = NewOverrides()
	}
	if overrides.om == nil {
		overrides.om = orderedmap.NewOrderedMap[string, *Override]()
	}
	defer overrides.mutex.Unlock()
	overrides.mutex.Lock()
	return overrides.om.Set(key, value)
}

// All returns an iterator that loops over all task key-value pairs.
// Range calls the provided function for each override in the map. The function
// receives the override's key and value as arguments. If the function returns
// an error, the iteration stops and the error is returned.
func (overrides *Overrides) All() iter.Seq2[string, *Override] {
	if overrides == nil || overrides.om == nil {
		return func(yield func(string, *Override) bool) {}
	}
	return overrides.om.AllFromFront()
}

// Keys returns an iterator that loops over all task keys.
func (overrides *Overrides) Keys() iter.Seq[string] {
	if overrides == nil || overrides.om == nil {
		return func(yield func(string) bool) {}
	}
	return overrides.om.Keys()
}

// Values returns an iterator that loops over all task values.
func (overrides *Overrides) Values() iter.Seq[*Override] {
	if overrides == nil || overrides.om == nil {
		return func(yield func(*Override) bool) {}
	}
	return overrides.om.Values()
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (overrides *Overrides) UnmarshalYAML(node *yaml.Node) error {
	if overrides == nil || overrides.om == nil {
		*overrides = *NewOverrides()
	}
	switch node.Kind {
	case yaml.MappingNode:
		// NOTE: orderedmap does not have an unmarshaler, so we have to decode
		// the map manually. We increment over 2 values at a time and assign
		// them as a key-value pair.
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			// Decode the value node into an Override struct
			var v Override
			if err := valueNode.Decode(&v); err != nil {
				return errors.NewTaskfileDecodeError(err, node)
			}

			// Set the override namespace
			v.Namespace = keyNode.Value

			// Add the override to the ordered map
			overrides.Set(keyNode.Value, &v)
		}
		return nil
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("overrides")
}

func (override *Override) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	case yaml.ScalarNode:
		var str string
		if err := node.Decode(&str); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		override.Taskfile = str
		// Overrides always flatten automatically
		override.Flatten = true
		return nil

	case yaml.MappingNode:
		var overrideTaskfile struct {
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
		if err := node.Decode(&overrideTaskfile); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		override.Taskfile = overrideTaskfile.Taskfile
		override.Dir = overrideTaskfile.Dir
		override.Optional = overrideTaskfile.Optional
		override.Internal = overrideTaskfile.Internal
		override.Aliases = overrideTaskfile.Aliases
		override.Excludes = overrideTaskfile.Excludes
		override.AdvancedImport = true
		override.Vars = overrideTaskfile.Vars
		// Overrides always flatten automatically, ignore the flatten setting from YAML
		override.Flatten = true
		override.Checksum = overrideTaskfile.Checksum
		return nil
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("override")
}

// DeepCopy creates a new instance of OverriddenTaskfile and copies
// data by value from the source struct.
func (override *Override) DeepCopy() *Override {
	if override == nil {
		return nil
	}
	return &Override{
		Namespace:      override.Namespace,
		Taskfile:       override.Taskfile,
		Dir:            override.Dir,
		Optional:       override.Optional,
		Internal:       override.Internal,
		Excludes:       deepcopy.Slice(override.Excludes),
		AdvancedImport: override.AdvancedImport,
		Vars:           override.Vars.DeepCopy(),
		Flatten:        override.Flatten,
		Aliases:        deepcopy.Slice(override.Aliases),
		Checksum:       override.Checksum,
	}
}
