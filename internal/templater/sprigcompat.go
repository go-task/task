package templater

import (
	"fmt"
	"reflect"

	"github.com/go-sprout/sprout"
	sproutmaps "github.com/go-sprout/sprout/registry/maps"
	"github.com/go-sprout/sprout/registry/slices"
)

// sprig passed the map or list to operate on as the *first* argument; sprout
// passes it *last* so that the function can be piped into. Taskfiles written
// against slim-sprig use the old order, and feeding them to sprout produces an
// opaque type error rather than a useful message.
//
// The wrappers below accept both orders, detecting the old one by the type of
// the first argument — the two positions never hold the same kind, so the test
// is unambiguous in every realistic case. When the old order is detected the
// call still succeeds, and a deprecation warning is logged.
//
// Only the ten functions whose argument order actually changed are wrapped.
// `dig`, `has`, `chunk` and `merge` kept theirs.

func sprigSignatureShims(handler sprout.Handler) sprout.FunctionMap {
	m := sproutmaps.NewRegistry()
	_ = m.LinkHandler(handler)
	s := slices.NewRegistry()
	_ = s.LinkHandler(handler)

	warn := func(name, oldSig, newSig string) {
		handler.Logger().
			With("function", name, "notice", "deprecated").
			Warn(fmt.Sprintf("Template function `%s` was called with the deprecated slim-sprig argument order `%s`; please use `%s` instead.", name, oldSig, newSig))
	}

	return sprout.FunctionMap{
		"get": func(args ...any) (any, error) {
			key, dict, err := mapArgs("get", `{{ get $dict "key" }}`, `{{ $dict | get "key" }}`, warn, args)
			if err != nil {
				return nil, err
			}
			return m.Get(key, dict)
		},
		"hasKey": func(args ...any) (any, error) {
			key, dict, err := mapArgs("hasKey", `{{ hasKey $dict "key" }}`, `{{ $dict | hasKey "key" }}`, warn, args)
			if err != nil {
				return nil, err
			}
			return m.HasKey(key, dict)
		},
		"unset": func(args ...any) (any, error) {
			key, dict, err := mapArgs("unset", `{{ unset $dict "key" }}`, `{{ $dict | unset "key" }}`, warn, args)
			if err != nil {
				return nil, err
			}
			return m.Unset(key, dict)
		},
		"set": func(args ...any) (any, error) {
			if len(args) != 3 {
				return nil, fmt.Errorf("set requires exactly 3 arguments, got %d", len(args))
			}
			if isMap(args[0]) {
				warn("set", `{{ set $dict "key" "value" }}`, `{{ $dict | set "key" "value" }}`)
				args = []any{args[1], args[2], args[0]}
			}
			key, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("set: key must be a string, got %T", args[0])
			}
			dict, ok := args[2].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("set: last argument must be a map, got %T", args[2])
			}
			return m.Set(key, args[1], dict)
		},
		// sprout's `dig` dropped sprig's default-value argument entirely:
		// it is `dig(keys..., dict)` where sprig had `dig(keys..., default,
		// dict)`. Both take strings in that position, so the two forms cannot
		// be told apart by type — Task keeps sprig's meaning, since that is
		// what every existing Taskfile was written against.
		"dig": func(args ...any) (any, error) {
			if len(args) < 3 {
				return m.Dig(args...)
			}
			dict, ok := args[len(args)-1].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("dig: last argument must be a map, got %T", args[len(args)-1])
			}
			fallback := args[len(args)-2]
			lookup := make([]any, 0, len(args)-1)
			lookup = append(lookup, args[:len(args)-2]...)
			out, err := m.Dig(append(lookup, dict)...)
			if err != nil || out == nil {
				return fallback, nil
			}
			return out, nil
		},
		"pick": func(args ...any) (any, error) {
			return m.Pick(rotateFirstToLast("pick", `{{ pick $dict "key" }}`, `{{ $dict | pick "key" }}`, warn, isMap, args)...)
		},
		"omit": func(args ...any) (any, error) {
			return m.Omit(rotateFirstToLast("omit", `{{ omit $dict "key" }}`, `{{ $dict | omit "key" }}`, warn, isMap, args)...)
		},
		"append": func(args ...any) (any, error) {
			v, list, err := listArgs("append", `{{ append $list "value" }}`, `{{ $list | append "value" }}`, warn, args)
			if err != nil {
				return nil, err
			}
			return s.Append(v, list)
		},
		"prepend": func(args ...any) (any, error) {
			v, list, err := listArgs("prepend", `{{ prepend $list "value" }}`, `{{ $list | prepend "value" }}`, warn, args)
			if err != nil {
				return nil, err
			}
			return s.Prepend(v, list)
		},
		"without": func(args ...any) (any, error) {
			return s.Without(rotateFirstToLast("without", `{{ without $list "value" }}`, `{{ $list | without "value" }}`, warn, isList, args)...)
		},
		"slice": func(args ...any) (any, error) {
			return s.Slice(rotateFirstToLast("slice", `{{ slice $list 1 3 }}`, `{{ $list | slice 1 3 }}`, warn, isList, args)...)
		},
	}
}

type warnFunc func(name, oldSig, newSig string)

// mapArgs resolves the (key, dict) pair of a two-argument map function written
// in either order.
func mapArgs(name, oldSig, newSig string, warn warnFunc, args []any) (string, map[string]any, error) {
	if len(args) != 2 {
		return "", nil, fmt.Errorf("%s requires exactly 2 arguments, got %d", name, len(args))
	}
	if isMap(args[0]) {
		warn(name, oldSig, newSig)
		args = []any{args[1], args[0]}
	}
	key, ok := args[0].(string)
	if !ok {
		return "", nil, fmt.Errorf("%s: key must be a string, got %T", name, args[0])
	}
	dict, ok := args[1].(map[string]any)
	if !ok {
		return "", nil, fmt.Errorf("%s: last argument must be a map, got %T", name, args[1])
	}
	return key, dict, nil
}

// listArgs resolves the (value, list) pair of a two-argument slice function
// written in either order.
func listArgs(name, oldSig, newSig string, warn warnFunc, args []any) (any, any, error) {
	if len(args) != 2 {
		return nil, nil, fmt.Errorf("%s requires exactly 2 arguments, got %d", name, len(args))
	}
	if isList(args[0]) && !isList(args[1]) {
		warn(name, oldSig, newSig)
		return args[1], args[0], nil
	}
	return args[0], args[1], nil
}

// rotateFirstToLast moves a leading target argument to the end, which is where
// sprout's variadic functions expect it.
func rotateFirstToLast(name, oldSig, newSig string, warn warnFunc, isTarget func(any) bool, args []any) []any {
	if len(args) < 2 || !isTarget(args[0]) || isTarget(args[len(args)-1]) {
		return args
	}
	warn(name, oldSig, newSig)
	rotated := make([]any, 0, len(args))
	rotated = append(rotated, args[1:]...)
	return append(rotated, args[0])
}

func isMap(v any) bool {
	_, ok := v.(map[string]any)
	return ok
}

func isList(v any) bool {
	if v == nil {
		return false
	}
	switch reflect.TypeOf(v).Kind() {
	case reflect.Slice, reflect.Array:
		return true
	default:
		return false
	}
}
