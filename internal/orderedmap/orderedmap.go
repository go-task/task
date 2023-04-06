package orderedmap

import (
	"fmt"

	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/internal/deepcopy"
)

// An OrderedMap is a wrapper around a regular map that maintains an ordered
// list of the map's keys. This allows you to run deterministic and ordered
// operations on the map such as printing/serializing/iterating.
type OrderedMap[K constraints.Ordered, V any] struct {
	s []K
	m map[K]V
}

// New will create a new OrderedMap of the given type and return it.
func New[K constraints.Ordered, V any]() OrderedMap[K, V] {
	return OrderedMap[K, V]{
		s: make([]K, 0),
		m: make(map[K]V),
	}
}

// FromMap will create a new OrderedMap from the given map. Since Golang maps
// are unordered, the order of the created OrderedMap will be random.
func FromMap[K constraints.Ordered, V any](m map[K]V) OrderedMap[K, V] {
	om := New[K, V]()
	om.m = m
	om.s = maps.Keys(m)
	return om
}

func FromMapWithOrder[K constraints.Ordered, V any](m map[K]V, order []K) OrderedMap[K, V] {
	om := New[K, V]()
	if len(m) != len(order) {
		panic("length of map and order must be equal")
	}
	om.m = m
	om.s = order
	for key := range om.m {
		if !slices.Contains(om.s, key) {
			panic("order keys must match map keys")
		}
	}
	return om
}

// Len will return the number of items in the map.
func (om *OrderedMap[K, V]) Len() int {
	return len(om.s)
}

// Set will set the value for a given key.
func (om *OrderedMap[K, V]) Set(key K, value V) {
	if om.m == nil {
		om.m = make(map[K]V)
	}
	if _, ok := om.m[key]; !ok {
		om.s = append(om.s, key)
	}
	om.m[key] = value
}

// Get will return the value for a given key.
// If the key does not exist, it will return the zero value of the value type.
func (om *OrderedMap[K, V]) Get(key K) V {
	value, ok := om.m[key]
	if !ok {
		var zero V
		return zero
	}
	return value
}

// Exists will return whether or not the given key exists.
func (om *OrderedMap[K, V]) Exists(key K) bool {
	_, ok := om.m[key]
	return ok
}

// Sort will sort the map.
func (om *OrderedMap[K, V]) Sort() {
	slices.Sort(om.s)
}

// SortFunc will sort the map using the given function.
func (om *OrderedMap[K, V]) SortFunc(less func(i, j K) bool) {
	slices.SortFunc(om.s, less)
}

// Keys will return a slice of the map's keys in order.
func (om *OrderedMap[K, V]) Keys() []K {
	return om.s
}

// Values will return a slice of the map's values in order.
func (om *OrderedMap[K, V]) Values() []V {
	var values []V
	for _, key := range om.s {
		values = append(values, om.m[key])
	}
	return values
}

// Range will iterate over the map and call the given function for each key/value.
func (om *OrderedMap[K, V]) Range(fn func(key K, value V) error) error {
	for _, key := range om.s {
		if err := fn(key, om.m[key]); err != nil {
			return err
		}
	}
	return nil
}

// Merge merges the given Vars into the caller one
func (om *OrderedMap[K, V]) Merge(other OrderedMap[K, V]) {
	// nolint: errcheck
	other.Range(func(key K, value V) error {
		om.Set(key, value)
		return nil
	})
}

func (om *OrderedMap[K, V]) DeepCopy() OrderedMap[K, V] {
	return OrderedMap[K, V]{
		s: deepcopy.Slice(om.s),
		m: deepcopy.Map(om.m),
	}
}

func (om *OrderedMap[K, V]) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	// Even numbers contain the keys
	// Odd numbers contain the values
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			// Decode the key
			keyNode := node.Content[i]
			var k K
			if err := keyNode.Decode(&k); err != nil {
				return err
			}

			// Decode the value
			valueNode := node.Content[i+1]
			var v V
			if err := valueNode.Decode(&v); err != nil {
				return err
			}

			// Set the key and value
			om.Set(k, v)
		}
		return nil
	}

	return fmt.Errorf("yaml: line %d: cannot unmarshal %s into variables", node.Line, node.ShortTag())
}
