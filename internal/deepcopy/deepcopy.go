package deepcopy

import (
	"reflect"

	"github.com/elliotchance/orderedmap/v3"
)

type Copier[T any] interface {
	DeepCopy() T
}

func Slice[T any](orig []T) []T {
	if orig == nil {
		return nil
	}
	c := make([]T, len(orig))
	for i, v := range orig {
		if copyable, ok := any(v).(Copier[T]); ok {
			c[i] = copyable.DeepCopy()
		} else {
			c[i] = v
		}
	}
	return c
}

func Map[K comparable, V any](orig map[K]V) map[K]V {
	if orig == nil {
		return nil
	}
	c := make(map[K]V, len(orig))
	for k, v := range orig {
		if copyable, ok := any(v).(Copier[V]); ok {
			c[k] = copyable.DeepCopy()
		} else {
			c[k] = v
		}
	}
	return c
}

func OrderedMap[K comparable, V any](orig *orderedmap.OrderedMap[K, V]) *orderedmap.OrderedMap[K, V] {
	if orig.Len() == 0 {
		return orderedmap.NewOrderedMap[K, V]()
	}
	c := orderedmap.NewOrderedMap[K, V]()
	for pair := orig.Front(); pair != nil; pair = pair.Next() {
		if copyable, ok := any(pair.Value).(Copier[V]); ok {
			c.Set(pair.Key, copyable.DeepCopy())
		} else {
			c.Set(pair.Key, pair.Value)
		}
	}
	return c
}

// TraverseStringsFunc runs the given function on every string in the given
// value by traversing it recursively. If the given value is a string, the
// function will run on a copy of the string and return it. If the value is a
// struct, map or a slice, the function will recursively call itself for each
// field or element of the struct, map or slice until all strings inside the
// struct or slice are replaced.
func TraverseStringsFunc[T any](v T, fn func(v string) (string, error)) (T, error) {
	original := reflect.ValueOf(v)
	if original.Kind() == reflect.Invalid || !original.IsValid() {
		return v, nil
	}
	copy := reflect.New(original.Type()).Elem()

	var traverseFunc func(copy, v reflect.Value) error
	traverseFunc = func(copy, v reflect.Value) error {
		switch v.Kind() {

		case reflect.Ptr:
			// Unwrap the pointer
			originalValue := v.Elem()
			// If the pointer is nil, do nothing
			if !originalValue.IsValid() {
				return nil
			}
			// Create an empty copy from the original value's type
			copy.Set(reflect.New(originalValue.Type()))
			// Unwrap the newly created pointer and call traverseFunc recursively
			if err := traverseFunc(copy.Elem(), originalValue); err != nil {
				return err
			}

		case reflect.Interface:
			// Unwrap the interface
			originalValue := v.Elem()
			if !originalValue.IsValid() {
				return nil
			}
			// Create an empty copy from the original value's type
			copyValue := reflect.New(originalValue.Type()).Elem()
			// Unwrap the newly created pointer and call traverseFunc recursively
			if err := traverseFunc(copyValue, originalValue); err != nil {
				return err
			}
			copy.Set(copyValue)

		case reflect.Struct:
			// Loop over each field and call traverseFunc recursively
			for i := range v.NumField() {
				if err := traverseFunc(copy.Field(i), v.Field(i)); err != nil {
					return err
				}
			}

		case reflect.Slice:
			// Create an empty copy from the original value's type
			copy.Set(reflect.MakeSlice(v.Type(), v.Len(), v.Cap()))
			// Loop over each element and call traverseFunc recursively
			for i := range v.Len() {
				if err := traverseFunc(copy.Index(i), v.Index(i)); err != nil {
					return err
				}
			}

		case reflect.Map:
			// Create an empty copy from the original value's type
			copy.Set(reflect.MakeMap(v.Type()))
			// Loop over each key
			for _, key := range v.MapKeys() {
				// Create a copy of each map index
				originalValue := v.MapIndex(key)
				if originalValue.IsNil() {
					continue
				}
				copyValue := reflect.New(originalValue.Type()).Elem()
				// Call traverseFunc recursively
				if err := traverseFunc(copyValue, originalValue); err != nil {
					return err
				}
				copy.SetMapIndex(key, copyValue)
			}

		case reflect.String:
			rv, err := fn(v.String())
			if err != nil {
				return err
			}
			copy.Set(reflect.ValueOf(rv))

		default:
			copy.Set(v)
		}

		return nil
	}

	if err := traverseFunc(copy, original); err != nil {
		return v, err
	}

	return copy.Interface().(T), nil
}
