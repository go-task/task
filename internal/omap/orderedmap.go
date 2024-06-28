package omap

import (
	"cmp"
	"fmt"
	"slices"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/internal/deepcopy"
)

type OrderedMap[K cmp.Ordered, V any] interface {
	Get(key K) V
	Set(key K, value V)
	Len() int
	Exists(key K) bool
	Range(fn func(key K, value V) error) error
	Values() []V
	Keys() []K
	Sort()
	SortFunc(less func(i, j K) int)
	DeepCopy() OrderedMap[K, V]
	Merge(om OrderedMap[K, V])
	yaml.Unmarshaler
}

// New will create a new OrderedMap of the given type and return it.
func New[K cmp.Ordered, V any]() OrderedMap[K, V] {
	return newOrderedMap[K, V]()
}

// FromMap will create a new OrderedMap from the given map. Since Golang maps
// are unordered, the order of the created OrderedMap will be random.
func FromMap[K cmp.Ordered, V any](m map[K]V) OrderedMap[K, V] {
	om := newOrderedMap[K, V]()
	for k, v := range m {
		om.Set(k, v)
	}
	return om
}

func FromMapWithOrder[K cmp.Ordered, V any](m map[K]V, order []K) OrderedMap[K, V] {
	if len(m) != len(order) {
		panic("length of map and order must be equal")
	}

	om := newOrderedMap[K, V]()
	om.m = m
	om.s = order
	for key := range om.m {
		if !slices.Contains(om.s, key) {
			panic("order keys must match map keys")
		}
	}
	return om
}

// An OrderedMap is a wrapper around a regular map that maintains an ordered
// list of the map's keys. This allows you to run deterministic and ordered
// operations on the map such as printing/serializing/iterating.
type orderedMap[K cmp.Ordered, V any] struct {
	mutex sync.RWMutex
	s     []K
	m     map[K]V
}

func newOrderedMap[K cmp.Ordered, V any]() *orderedMap[K, V] {
	return &orderedMap[K, V]{
		s: make([]K, 0),
		m: make(map[K]V),
	}
}

// Len will return the number of items in the map.
func (om *orderedMap[K, V]) Len() (l int) {
	om.mutex.RLock()
	l = len(om.s)
	om.mutex.RUnlock()

	return
}

// Set will set the value for a given key.
func (om *orderedMap[K, V]) Set(key K, value V) {
	om.mutex.Lock()
	if _, ok := om.m[key]; !ok {
		om.s = append(om.s, key)
	}
	om.m[key] = value
	om.mutex.Unlock()
}

// Get will return the value for a given key.
// If the key does not exist, it will return the zero value of the value type.
func (om *orderedMap[K, V]) Get(key K) V {
	om.mutex.RLock()
	value, ok := om.m[key]
	om.mutex.RUnlock()
	if !ok {
		var zero V
		return zero
	}
	return value
}

// Exists will return whether or not the given key exists.
func (om *orderedMap[K, V]) Exists(key K) bool {
	om.mutex.RLock()
	_, ok := om.m[key]
	om.mutex.RUnlock()
	return ok
}

// Sort will sort the map.
func (om *orderedMap[K, V]) Sort() {
	om.mutex.Lock()
	slices.Sort(om.s)
	om.mutex.Unlock()
}

// SortFunc will sort the map using the given function.
func (om *orderedMap[K, V]) SortFunc(less func(i, j K) int) {
	om.mutex.Lock()
	slices.SortFunc(om.s, less)
	om.mutex.Unlock()
}

// Keys will return a slice of the map's keys in order.
func (om *orderedMap[K, V]) Keys() []K {
	om.mutex.RLock()
	keys := deepcopy.Slice(om.s)
	om.mutex.RUnlock()

	return keys
}

// Values will return a slice of the map's values in order.
func (om *orderedMap[K, V]) Values() []V {
	var values []V

	om.mutex.RLock()
	for _, key := range om.s {
		values = append(values, om.Get(key))
	}
	om.mutex.RUnlock()

	return values
}

// Range will iterate over the map and call the given function for each key/value.
func (om *orderedMap[K, V]) Range(fn func(key K, value V) error) error {
	keys := om.Keys()
	for _, key := range keys {
		if err := fn(key, om.Get(key)); err != nil {
			return err
		}
	}
	return nil
}

// Merge merges the given Vars into the caller one
func (om *orderedMap[K, V]) Merge(other OrderedMap[K, V]) {
	// nolint: errcheck
	other.Range(func(key K, value V) error {
		om.Set(key, value)
		return nil
	})
}

func (om *orderedMap[K, V]) DeepCopy() OrderedMap[K, V] {
	om.mutex.RLock()
	o := orderedMap[K, V]{
		s: deepcopy.Slice(om.s),
		m: deepcopy.Map(om.m),
	}
	om.mutex.RUnlock()

	return &o
}

func (om *orderedMap[K, V]) UnmarshalYAML(node *yaml.Node) error {
	if om == nil {
		*om = orderedMap[K, V]{
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
	}

	return fmt.Errorf("yaml: line %d: cannot unmarshal %s into variables", node.Line, node.ShortTag())
}
