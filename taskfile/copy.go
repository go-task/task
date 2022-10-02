package taskfile

import "golang.org/x/exp/constraints"

func deepCopySlice[T any](orig []T) []T {
	if orig == nil {
		return nil
	}
	c := make([]T, len(orig))
	copy(c, orig)
	return c
}

func deepCopyMap[K constraints.Ordered, V any](orig map[K]V) map[K]V {
	if orig == nil {
		return nil
	}
	c := make(map[K]V, len(orig))
	for k, v := range orig {
		c[k] = v
	}
	return c
}
