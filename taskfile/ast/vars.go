package ast

import (
	"sync"

	"github.com/elliotchance/orderedmap/v2"
	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/deepcopy"
)

type (
	// Vars is an ordered map of variable names to values.
	Vars struct {
		om    *orderedmap.OrderedMap[string, Var]
		mutex sync.RWMutex
	}
	// A VarElement is a key-value pair that is used for initializing a Vars
	// structure.
	VarElement orderedmap.Element[string, Var]
)

// NewVars creates a new instance of Vars and initializes it with the provided
// set of elements, if any. The elements are added in the order they are passed.
func NewVars(els ...*VarElement) *Vars {
	vars := &Vars{
		om: orderedmap.NewOrderedMap[string, Var](),
	}
	for _, el := range els {
		vars.Set(el.Key, el.Value)
	}
	return vars
}

// Len returns the number of variables in the Vars map.
func (vars *Vars) Len() int {
	if vars == nil || vars.om == nil {
		return 0
	}
	defer vars.mutex.RUnlock()
	vars.mutex.RLock()
	return vars.om.Len()
}

// Get returns the value the the variable with the provided key and a boolean
// that indicates if the value was found or not. If the value is not found, the
// returned variable is a zero value and the bool is false.
func (vars *Vars) Get(key string) (Var, bool) {
	if vars == nil || vars.om == nil {
		return Var{}, false
	}
	defer vars.mutex.RUnlock()
	vars.mutex.RLock()
	return vars.om.Get(key)
}

// Set sets the value of the variable with the provided key to the provided
// value. If the variable already exists, its value is updated. If the variable
// does not exist, it is created.
func (vars *Vars) Set(key string, value Var) bool {
	if vars == nil {
		vars = NewVars()
	}
	if vars.om == nil {
		vars.om = orderedmap.NewOrderedMap[string, Var]()
	}
	defer vars.mutex.Unlock()
	vars.mutex.Lock()
	return vars.om.Set(key, value)
}

// Range calls the provided function for each variable in the map. The function
// receives the variable's key and value as arguments. If the function returns
// an error, the iteration stops and the error is returned.
func (vars *Vars) Range(f func(k string, v Var) error) error {
	if vars == nil || vars.om == nil {
		return nil
	}
	for pair := vars.om.Front(); pair != nil; pair = pair.Next() {
		if err := f(pair.Key, pair.Value); err != nil {
			return err
		}
	}
	return nil
}

// ToCacheMap converts Vars to an unordered map containing only the static
// variables
func (vars *Vars) ToCacheMap() (m map[string]any) {
	defer vars.mutex.RUnlock()
	vars.mutex.RLock()
	m = make(map[string]any, vars.Len())
	for pair := vars.om.Front(); pair != nil; pair = pair.Next() {
		if pair.Value.Sh != nil && *pair.Value.Sh != "" {
			// Dynamic variable is not yet resolved; trigger
			// <no value> to be used in templates.
			return nil
		}
		if pair.Value.Live != nil {
			m[pair.Key] = pair.Value.Live
		} else {
			m[pair.Key] = pair.Value.Value
		}
	}
	return
}

// Merge loops over other and merges it values with the variables in vars. If
// the include parameter is not nil and its it is an advanced import, the
// directory is set set to the value of the include parameter.
func (vars *Vars) Merge(other *Vars, include *Include) {
	if vars == nil || vars.om == nil || other == nil {
		return
	}
	defer other.mutex.RUnlock()
	other.mutex.RLock()
	for pair := other.om.Front(); pair != nil; pair = pair.Next() {
		if include != nil && include.AdvancedImport {
			pair.Value.Dir = include.Dir
		}
		vars.om.Set(pair.Key, pair.Value)
	}
}

func (vs *Vars) DeepCopy() *Vars {
	if vs == nil {
		return nil
	}
	defer vs.mutex.RUnlock()
	vs.mutex.RLock()
	return &Vars{
		om: deepcopy.OrderedMap(vs.om),
	}
}

func (vs *Vars) UnmarshalYAML(node *yaml.Node) error {
	if vs == nil || vs.om == nil {
		*vs = *NewVars()
	}
	vs.om = orderedmap.NewOrderedMap[string, Var]()
	switch node.Kind {
	case yaml.MappingNode:
		// NOTE: orderedmap does not have an unmarshaler, so we have to decode
		// the map manually. We increment over 2 values at a time and assign
		// them as a key-value pair.
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			// Decode the value node into a Task struct
			var v Var
			if err := valueNode.Decode(&v); err != nil {
				return errors.NewTaskfileDecodeError(err, node)
			}

			// Add the task to the ordered map
			vs.Set(keyNode.Value, v)
		}
		return nil
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("vars")
}
