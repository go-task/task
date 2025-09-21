package ast

import (
	"iter"

	"github.com/elliotchance/orderedmap/v3"
	"go.yaml.in/yaml/v4"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/deepcopy"
)

type (
	// Matrix is an ordered map of variable names to arrays of values.
	Matrix struct {
		om *orderedmap.OrderedMap[string, *MatrixRow]
	}
	// A MatrixElement is a key-value pair that is used for initializing a
	// Matrix structure.
	MatrixElement orderedmap.Element[string, *MatrixRow]
	// A MatrixRow list of values for a matrix key or a reference to another
	// variable.
	MatrixRow struct {
		Ref   string
		Value []any
	}
)

func NewMatrix(els ...*MatrixElement) *Matrix {
	matrix := &Matrix{
		om: orderedmap.NewOrderedMap[string, *MatrixRow](),
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

func (matrix *Matrix) Get(key string) (*MatrixRow, bool) {
	if matrix == nil || matrix.om == nil {
		return nil, false
	}
	return matrix.om.Get(key)
}

func (matrix *Matrix) Set(key string, value *MatrixRow) bool {
	if matrix == nil {
		matrix = NewMatrix()
	}
	if matrix.om == nil {
		matrix.om = orderedmap.NewOrderedMap[string, *MatrixRow]()
	}
	return matrix.om.Set(key, value)
}

// All returns an iterator that loops over all task key-value pairs.
func (matrix *Matrix) All() iter.Seq2[string, *MatrixRow] {
	if matrix == nil || matrix.om == nil {
		return func(yield func(string, *MatrixRow) bool) {}
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
func (matrix *Matrix) Values() iter.Seq[*MatrixRow] {
	if matrix == nil || matrix.om == nil {
		return func(yield func(*MatrixRow) bool) {}
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

			switch valueNode.Kind {
			case yaml.SequenceNode:
				// Decode the value node into a Matrix struct
				var v []any
				if err := valueNode.Decode(&v); err != nil {
					return errors.NewTaskfileDecodeError(err, node)
				}

				// Add the row to the ordered map
				matrix.Set(keyNode.Value, &MatrixRow{
					Value: v,
				})

			case yaml.MappingNode:
				// Decode the value node into a Matrix struct
				var refStruct struct {
					Ref string
				}
				if err := valueNode.Decode(&refStruct); err != nil {
					return errors.NewTaskfileDecodeError(err, node)
				}

				// Add the reference to the ordered map
				matrix.Set(keyNode.Value, &MatrixRow{
					Ref: refStruct.Ref,
				})

			default:
				return errors.NewTaskfileDecodeError(nil, node).WithMessage("matrix values must be an array or a reference")
			}
		}
		return nil
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("matrix")
}
