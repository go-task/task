package ast

import (
	"iter"

	"github.com/elliotchance/orderedmap/v3"
	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/deepcopy"
)

type Matrix struct {
	om *orderedmap.OrderedMap[string, []any]
}

type MatrixElement orderedmap.Element[string, []any]

func NewMatrix(els ...*MatrixElement) *Matrix {
	matrix := &Matrix{
		om: orderedmap.NewOrderedMap[string, []any](),
	}
	for _, el := range els {
		matrix.Set(el.Key, el.Value)
	}
	return matrix
}

func (matrix *Matrix) Len() int {
	if matrix == nil || matrix.om == nil {
		return 0
	}
	return matrix.om.Len()
}

func (matrix *Matrix) Get(key string) ([]any, bool) {
	if matrix == nil || matrix.om == nil {
		return nil, false
	}
	return matrix.om.Get(key)
}

func (matrix *Matrix) Set(key string, value []any) bool {
	if matrix == nil {
		matrix = NewMatrix()
	}
	if matrix.om == nil {
		matrix.om = orderedmap.NewOrderedMap[string, []any]()
	}
	return matrix.om.Set(key, value)
}

// All returns an iterator that loops over all task key-value pairs.
func (matrix *Matrix) All() iter.Seq2[string, []any] {
	if matrix == nil || matrix.om == nil {
		return func(yield func(string, []any) bool) {}
	}
	return matrix.om.AllFromFront()
}

// Keys returns an iterator that loops over all task keys.
func (matrix *Matrix) Keys() iter.Seq[string] {
	if matrix == nil || matrix.om == nil {
		return func(yield func(string) bool) {}
	}
	return matrix.om.Keys()
}

// Values returns an iterator that loops over all task values.
func (matrix *Matrix) Values() iter.Seq[[]any] {
	if matrix == nil || matrix.om == nil {
		return func(yield func([]any) bool) {}
	}
	return matrix.om.Values()
}

func (matrix *Matrix) DeepCopy() *Matrix {
	if matrix == nil {
		return nil
	}
	return &Matrix{
		om: deepcopy.OrderedMap(matrix.om),
	}
}

func (matrix *Matrix) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.MappingNode:
		// NOTE: orderedmap does not have an unmarshaler, so we have to decode
		// the map manually. We increment over 2 values at a time and assign
		// them as a key-value pair.
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			// Decode the value node into a Matrix struct
			var v []any
			if err := valueNode.Decode(&v); err != nil {
				return errors.NewTaskfileDecodeError(err, node)
			}

			// Add the task to the ordered map
			matrix.Set(keyNode.Value, v)
		}
		return nil
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("matrix")
}
