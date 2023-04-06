package deepcopy

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
