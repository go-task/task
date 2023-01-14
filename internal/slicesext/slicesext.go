package slicesext

import (
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

func UniqueJoin[T constraints.Ordered](ss ...[]T) []T {
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
