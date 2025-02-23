package slicesext

import (
	"cmp"
	"slices"
)

func UniqueJoin[T cmp.Ordered](ss ...[]T) []T {
	var length int
	for _, s := range ss {
		length += len(s)
	}
	r := make([]T, length)
	var i int
	for _, s := range ss {
		i += copy(r[i:], s)
	}
	slices.Sort(r)
	return slices.Compact(r)
}

func Convert[T, U any](s []T, f func(T) U) []U {
	// Create a new slice with the same length as the input slice
	result := make([]U, len(s))

	// Convert each element using the provided function
	for i, v := range s {
		result[i] = f(v)
	}

	return result
}
