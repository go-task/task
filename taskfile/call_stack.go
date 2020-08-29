package taskfile

import (
	"reflect"
)

// CallStack is a stack of calls that happen one after another.
type CallStack []Call

// Contains returns true if the given needle call occurs in the call stack.
func (stack CallStack) Contains(needle Call) bool {
	for _, call := range stack {
		if call.Task == needle.Task && reflect.DeepEqual(call.Vars, needle.Vars) {
			return true
		}
	}
	return false
}

// Add adds the given call to the given call stack.
// It returns a new call stack to ensure that the original one is not modified.
func (stack CallStack) Add(call Call) CallStack {
	cpy := make(CallStack, 0, len(stack))
	cpy = append(cpy, stack...)
	cpy = append(cpy, call)
	return cpy
}
