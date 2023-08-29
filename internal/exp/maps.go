// This package is intended as a place to copy functions from the
// golang.org/x/exp package. Copying these functions allows us to rely on our
// own code instead of an external package that may change unpredictably in the
// future.
//
// It also prevents problems with transitive dependencies whereby a
// package that imports Task (and therefore our version of golang.org/x/exp)
// cannot import a different version of golang.org/x/exp.
//
// Finally, it serves as a place to track functions that may be able to be
// removed in the future if they are added to the standard library. This is also
// why this package is under the internal directory since these functions are
// not intended to be used outside of Task.
package exp

import "cmp"

// Keys is a copy of https://pkg.go.dev/golang.org/x/exp@v0.0.0-20240103183307-be819d1f06fc/maps#Keys.
// This is not yet included in the standard library. See https://github.com/golang/go/issues/61538.
func Keys[K cmp.Ordered, V any](m map[K]V) []K {
	var keys []K
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}
