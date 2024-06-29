package omap

import (
	"cmp"
	"fmt"
	"slices"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/internal/deepcopy"
	"github.com/go-task/task/v3/internal/exp"
)

// An OrderedMap is a wrapper around a regular map that maintains an ordered
// list of the map's keys. This allows you to run deterministic and ordered
// operations on the map such as printing/serializing/iterating.
type OrderedMap[K cmp.Ordered, V any] struct {
	mutex sync.RWMutex
	s     []K
	m     map[K]V
}

// New will create a new OrderedMap of the given type and return it.
func New[K cmp.Ordered, V any]() OrderedMap[K, V] {
	return OrderedMap[K, V]{}
}

// FromMap will create a new OrderedMap from the given map. Since Golang maps
// are unordered, the order of the created OrderedMap will be random.
func FromMap[K cmp.Ordered, V any](m map[K]V) OrderedMap[K, V] {
	mm := deepcopy.Map(m)
	ms := exp.Keys(mm)
	return OrderedMap[K, V]{
		m: mm,
		s: ms,
	}
}

func FromMapWithOrder[K cmp.Ordered, V any](m map[K]V, order []K) OrderedMap[K, V] {
	if len(m) != len(order) {
		panic("length of map and order must be equal")
	}

	for key := range m {
		if !slices.Contains(order, key) {
			panic("order keys must match map keys")
		}
	}

	return OrderedMap[K, V]{
		s: slices.Clone(order),
		m: deepcopy.Map(m),
	}
}

// Len will return the number of items in the map.
func (om *OrderedMap[K, V]) Len() (l int) {
	om.mutex.RLock()
	l = len(om.s)
	om.mutex.RUnlock()

	return
}

// Set will set the value for a given key.
func (om *OrderedMap[K, V]) Set(key K, value V) {
	om.mutex.Lock()
	if om.m == nil {
		om.m = make(map[K]V)
	}
	if _, ok := om.m[key]; !ok {
		om.s = append(om.s, key)
	}
	om.m[key] = value
	om.mutex.Unlock()
}

// Get will return the value for a given key.
// If the key does not exist, it will return the zero value of the value type.
func (om *OrderedMap[K, V]) Get(key K) V {
	om.mutex.RLock()
	defer om.mutex.RUnlock()

	if om.m == nil {
		var zero V
		return zero
	}
	value, ok := om.m[key]
	if !ok {
		var zero V
		return zero
	}
	return value
}

// Exists will return whether or not the given key exists.
func (om *OrderedMap[K, V]) Exists(key K) bool {
	om.mutex.RLock()
	_, ok := om.m[key]
	om.mutex.RUnlock()
	return ok
}

// Sort will sort the map.
func (om *OrderedMap[K, V]) Sort() {
	om.mutex.Lock()
	slices.Sort(om.s)
	om.mutex.Unlock()
}

// SortFunc will sort the map using the given function.
func (om *OrderedMap[K, V]) SortFunc(less func(i, j K) int) {
	om.mutex.Lock()
	slices.SortFunc(om.s, less)
	om.mutex.Unlock()
}

// Keys will return a slice of the map's keys in order.
func (om *OrderedMap[K, V]) Keys() []K {
	om.mutex.RLock()
	keys := deepcopy.Slice(om.s)
	om.mutex.RUnlock()

	return keys
}

// Values will return a slice of the map's values in order.
func (om *OrderedMap[K, V]) Values() []V {
	om.mutex.RLock()
	values := make([]V, 0, len(om.m))
	for _, key := range om.s {
		values = append(values, om.m[key])
	}
	om.mutex.RUnlock()

	return values
}

// Range will iterate over the map and call the given function for each key/value.
func (om *OrderedMap[K, V]) Range(fn func(key K, value V) error) error {
	keys := om.Keys()
	for _, key := range keys {
		if err := fn(key, om.Get(key)); err != nil {
			return err
		}
	}
	return nil
}

// Merge merges the given Vars into the caller one
func (om *OrderedMap[K, V]) Merge(other *OrderedMap[K, V]) {
	_ = other.Range(func(key K, value V) error {
		om.Set(key, value)
		return nil
	})
}

func (om *OrderedMap[K, V]) DeepCopy() OrderedMap[K, V] {
	om.mutex.RLock()
	s := deepcopy.Slice(om.s)
	m := deepcopy.Map(om.m)
	om.mutex.RUnlock()

	return OrderedMap[K, V]{
		s: s,
		m: m,
	}
}

func (om *OrderedMap[K, V]) UnmarshalYAML(node *yaml.Node) error {
	if om == nil {
		*om = OrderedMap[K, V]{
			m: make(map[K]V),
			s: make([]K, 0),
		}
	}

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
	default:
		return fmt.Errorf("yaml: line %d: cannot unmarshal %s into variables", node.Line, node.ShortTag())
	}
}
