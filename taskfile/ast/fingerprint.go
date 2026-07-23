package ast

import (
	"reflect"
	"strings"
)

// ReferencesFingerprintVar reports whether the task references the variable
// produced by the given fingerprint method in fields compiled after it.
func (t *Task) ReferencesFingerprintVar(kind string) bool {
	name := strings.ToUpper(kind)
	if t == nil || name == "" {
		return false
	}

	for _, status := range t.Status {
		if stringReferencesFingerprintVar(status, name) {
			return true
		}
	}
	for _, cmd := range t.Cmds {
		if cmdReferencesFingerprintVar(cmd, name) {
			return true
		}
	}
	for _, dep := range t.Deps {
		if depReferencesFingerprintVar(dep, name) {
			return true
		}
	}
	for _, precondition := range t.Preconditions {
		if preconditionReferencesFingerprintVar(precondition, name) {
			return true
		}
	}
	return false
}

func cmdReferencesFingerprintVar(cmd *Cmd, name string) bool {
	if cmd == nil {
		return false
	}
	return stringReferencesFingerprintVar(cmd.Cmd, name) ||
		stringReferencesFingerprintVar(cmd.Task, name) ||
		stringReferencesFingerprintVar(cmd.If, name) ||
		forReferencesFingerprintVar(cmd.For, name) ||
		varsReferenceFingerprintVar(cmd.Vars, name)
}

func depReferencesFingerprintVar(dep *Dep, name string) bool {
	if dep == nil {
		return false
	}
	return stringReferencesFingerprintVar(dep.Task, name) ||
		forReferencesFingerprintVar(dep.For, name) ||
		varsReferenceFingerprintVar(dep.Vars, name)
}

func preconditionReferencesFingerprintVar(precondition *Precondition, name string) bool {
	if precondition == nil {
		return false
	}
	return stringReferencesFingerprintVar(precondition.Sh, name) ||
		stringReferencesFingerprintVar(precondition.Msg, name)
}

func forReferencesFingerprintVar(f *For, name string) bool {
	if f == nil {
		return false
	}
	if valueReferencesFingerprintVar(f.List, name) {
		return true
	}
	for _, row := range f.Matrix.All() {
		if row != nil && (stringReferencesFingerprintVar(row.Ref, name) ||
			valueReferencesFingerprintVar(row.Value, name)) {
			return true
		}
	}
	return false
}

func varsReferenceFingerprintVar(vars *Vars, name string) bool {
	for _, v := range vars.All() {
		if valueReferencesFingerprintVar(v.Value, name) ||
			valueReferencesFingerprintVar(v.Live, name) ||
			stringPointerReferencesFingerprintVar(v.Sh, name) ||
			stringReferencesFingerprintVar(v.Ref, name) ||
			stringReferencesFingerprintVar(v.Dir, name) {
			return true
		}
	}
	return false
}

func valueReferencesFingerprintVar(value any, name string) bool {
	if value == nil {
		return false
	}
	if s, ok := value.(string); ok {
		return stringReferencesFingerprintVar(s, name)
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return false
		}
		return valueReferencesFingerprintVar(rv.Elem().Interface(), name)
	case reflect.Array, reflect.Slice:
		for i := 0; i < rv.Len(); i++ {
			if valueReferencesFingerprintVar(rv.Index(i).Interface(), name) {
				return true
			}
		}
	case reflect.Map:
		for _, key := range rv.MapKeys() {
			if valueReferencesFingerprintVar(key.Interface(), name) ||
				valueReferencesFingerprintVar(rv.MapIndex(key).Interface(), name) {
				return true
			}
		}
	}
	return false
}

func stringPointerReferencesFingerprintVar(s *string, name string) bool {
	return s != nil && stringReferencesFingerprintVar(*s, name)
}

func stringReferencesFingerprintVar(s string, name string) bool {
	return strings.Contains(s, name)
}
