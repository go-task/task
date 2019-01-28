// Copyright (c) 2018, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package expand

import (
	"runtime"
	"sort"
	"strings"
)

// Environ is the base interface for a shell's environment, allowing it to fetch
// variables by name and to iterate over all the currently set variables.
type Environ interface {
	// Get retrieves a variable by its name. To check if the variable is
	// set, use Variable.IsSet.
	Get(name string) Variable

	// Each iterates over all the currently set variables, calling the
	// supplied function on each variable. Iteration is stopped if the
	// function returns false.
	//
	// The names used in the calls aren't required to be unique or sorted.
	// If a variable name appears twice, the latest occurrence takes
	// priority.
	//
	// Each is required to forward exported variables when executing
	// programs.
	Each(func(name string, vr Variable) bool)
}

// WriteEnviron is an extension on Environ that supports modifying and deleting
// variables.
type WriteEnviron interface {
	Environ
	// Set sets a variable by name. If !vr.IsSet(), the variable is being
	// unset; otherwise, the variable is being replaced.
	//
	// It is the implementation's responsibility to handle variable
	// attributes correctly. For example, changing an exported variable's
	// value does not unexport it, and overwriting a name reference variable
	// should modify its target.
	Set(name string, vr Variable)
}

// Variable describes a shell variable, which can have a number of attributes
// and a value.
//
// A Variable is unset if its Value field is untyped nil, which can be checked
// via Variable.IsSet. The zero value of a Variable is thus a valid unset
// variable.
//
// If a variable is set, its Value field will be a []string if it is an indexed
// array, a map[string]string if it's an associative array, or a string
// otherwise.
type Variable struct {
	Local    bool
	Exported bool
	ReadOnly bool
	NameRef  bool        // if true, Value must be string
	Value    interface{} // string, []string, or map[string]string
}

// IsSet returns whether the variable is set. An empty variable is set, but an
// undeclared variable is not.
func (v Variable) IsSet() bool {
	return v.Value != nil
}

// String returns the variable's value as a string. In general, this only makes
// sense if the variable has a string value or no value at all.
func (v Variable) String() string {
	switch x := v.Value.(type) {
	case string:
		return x
	case []string:
		if len(x) > 0 {
			return x[0]
		}
	case map[string]string:
		// nothing to do
	}
	return ""
}

// maxNameRefDepth defines the maximum number of times to follow references when
// resolving a variable. Otherwise, simple name reference loops could crash a
// program quite easily.
const maxNameRefDepth = 100

// Resolve follows a number of nameref variables, returning the last reference
// name that was followed and the variable that it points to.
func (v Variable) Resolve(env Environ) (string, Variable) {
	name := ""
	for i := 0; i < maxNameRefDepth; i++ {
		if !v.NameRef {
			return name, v
		}
		name = v.Value.(string)
		v = env.Get(name)
	}
	return name, Variable{}
}

// FuncEnviron wraps a function mapping variable names to their string values,
// and implements Environ. Empty strings returned by the function will be
// treated as unset variables. All variables will be exported.
//
// Note that the returned Environ's Each method will be a no-op.
func FuncEnviron(fn func(string) string) Environ {
	return funcEnviron(fn)
}

type funcEnviron func(string) string

func (f funcEnviron) Get(name string) Variable {
	value := f(name)
	if value == "" {
		return Variable{}
	}
	return Variable{Exported: true, Value: value}
}

func (f funcEnviron) Each(func(name string, vr Variable) bool) {}

// ListEnviron returns an Environ with the supplied variables, in the form
// "key=value". All variables will be exported.
//
// On Windows, where environment variable names are case-insensitive, the
// resulting variable names will all be uppercase.
func ListEnviron(pairs ...string) Environ {
	return listEnvironWithUpper(runtime.GOOS == "windows", pairs...)
}

// listEnvironWithUpper implements ListEnviron, but letting the tests specify
// whether to uppercase all names or not.
func listEnvironWithUpper(upper bool, pairs ...string) Environ {
	list := append([]string{}, pairs...)
	if upper {
		// Uppercase before sorting, so that we can remove duplicates
		// without the need for linear search nor a map.
		for i, s := range list {
			if sep := strings.IndexByte(s, '='); sep > 0 {
				list[i] = strings.ToUpper(s[:sep]) + s[sep:]
			}
		}
	}
	sort.Strings(list)
	last := ""
	for i := 0; i < len(list); {
		s := list[i]
		sep := strings.IndexByte(s, '=')
		if sep <= 0 {
			// invalid element; remove it
			list = append(list[:i], list[i+1:]...)
			continue
		}
		name := s[:sep]
		if last == name {
			// duplicate; the last one wins
			list = append(list[:i-1], list[i:]...)
			continue
		}
		last = name
		i++
	}
	return listEnviron(list)
}

type listEnviron []string

func (l listEnviron) Get(name string) Variable {
	// TODO: binary search
	prefix := name + "="
	for _, pair := range l {
		if val := strings.TrimPrefix(pair, prefix); val != pair {
			return Variable{Exported: true, Value: val}
		}
	}
	return Variable{}
}

func (l listEnviron) Each(fn func(name string, vr Variable) bool) {
	for _, pair := range l {
		i := strings.IndexByte(pair, '=')
		if i < 0 {
			// can't happen; see above
			panic("expand.listEnviron: did not expect malformed name-value pair: " + pair)
		}
		name, value := pair[:i], pair[i+1:]
		if !fn(name, Variable{Exported: true, Value: value}) {
			return
		}
	}
}
