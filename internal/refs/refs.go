// Package refs resolves the `ref` fields of a Taskfile into concrete values.
package refs

import (
	"fmt"

	"github.com/go-task/task/v3/internal/slicesext"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile/ast"
)

// Declared lists resolve to a []any, but `keys` and `splitList` return a []string.
func AsList(v any) ([]any, bool) {
	switch value := v.(type) {
	case []any:
		return value, true
	case []string:
		return slicesext.AsAny(value), true
	case []int:
		return slicesext.AsAny(value), true
	}
	return nil, false
}

func ResolveEnums(requires *ast.Requires, cache *templater.Cache) error {
	if requires == nil || len(requires.Vars) == 0 {
		return nil
	}
	for _, v := range requires.Vars {
		if v.Enum == nil || v.Enum.Ref == "" {
			continue
		}
		resolved := templater.ResolveRef(v.Enum.Ref, cache)
		if cache.Err() != nil {
			return cache.Err()
		}
		arr, ok := AsList(resolved)
		if !ok {
			return fmt.Errorf("enum reference %q must resolve to a list", v.Enum.Ref)
		}
		strValues := make([]string, 0, len(arr))
		for _, item := range arr {
			s, ok := item.(string)
			if !ok {
				return fmt.Errorf("enum reference %q must contain only strings", v.Enum.Ref)
			}
			strValues = append(strValues, s)
		}
		v.Enum.Value = strValues
	}
	return nil
}

// A ref depending on dynamic vars may not resolve: the copy then has no enum
// value, which the interactive prompter treats as free-form input.
func ResolveEnum(v *ast.VarsWithValidation, vars *ast.Vars) *ast.VarsWithValidation {
	if v.Enum == nil || v.Enum.Ref == "" || len(v.Enum.Value) > 0 {
		return v
	}
	vCopy := v.DeepCopy()
	cache := &templater.Cache{Vars: vars}
	_ = ResolveEnums(&ast.Requires{Vars: []*ast.VarsWithValidation{vCopy}}, cache)
	return vCopy
}
