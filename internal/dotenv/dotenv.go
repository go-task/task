// Package dotenv reads environment files in declaration order.
package dotenv

import (
	"os"

	"github.com/elliotchance/orderedmap/v3"
)

// Read reads a dotenv file, preserving the order of its variable declarations.
// Repeated keys retain their original position and take their last value.
func Read(filename string) (*orderedmap.OrderedMap[string, string], error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	vars := orderedmap.NewOrderedMap[string, string]()
	if err := parseBytes(data, vars); err != nil {
		return nil, err
	}
	return vars, nil
}
