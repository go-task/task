package templater

import (
	"bytes"
	"maps"
	"reflect"
	"strings"
	"text/template"

	"github.com/go-task/task/v3/taskfile/ast"
)

// Cache is a help struct that allow us to call "replaceX" funcs multiple
// times, without having to check for error each time. The first error that
// happen will be assigned to r.err, and consecutive calls to funcs will just
// return the zero value.
type Cache struct {
	Vars *ast.Vars

	cacheMap map[string]any
	err      error
}

func (r *Cache) ResetCache() {
	r.cacheMap = r.Vars.ToCacheMap()
}

func (r *Cache) Err() error {
	return r.err
}

func Replace[T any](v T, cache *Cache) T {
	return ReplaceWithExtra(v, cache, nil)
}

func ReplaceWithExtra[T any](v T, cache *Cache, extra map[string]any) T {
	// If there is already an error, do nothing
	if cache.err != nil {
		return v
	}

	// Initialize the cache map if it's not already initialized
	if cache.cacheMap == nil {
		cache.cacheMap = cache.Vars.ToCacheMap()
	}

	// Create a copy of the cache map to avoid editing the original
	// If there is extra data, merge it with the cache map
	data := maps.Clone(cache.cacheMap)
	if extra != nil {
		maps.Copy(data, extra)
	}

	original := reflect.ValueOf(&v)
	copy := reflect.New(original.Type()).Elem()

	// Replace the variables in the value
	if err := replaceValue(copy, original, data); err != nil {
		cache.err = err
		return v
	}

	return copy.Elem().Interface().(T)
}

func ReplaceGlobs(globs []*ast.Glob, cache *Cache) []*ast.Glob {
	if cache.err != nil || len(globs) == 0 {
		return nil
	}

	new := make([]*ast.Glob, len(globs))
	for i, g := range globs {
		new[i] = &ast.Glob{
			Glob:   Replace(g.Glob, cache),
			Negate: g.Negate,
		}
	}
	return new
}

func ReplaceVars(vars *ast.Vars, cache *Cache) *ast.Vars {
	return ReplaceVarsWithExtra(vars, cache, nil)
}

func ReplaceVarsWithExtra(vars *ast.Vars, cache *Cache, extra map[string]any) *ast.Vars {
	if cache.err != nil || vars.Len() == 0 {
		return nil
	}

	var newVars ast.Vars
	_ = vars.Range(func(k string, v ast.Var) error {
		var newVar ast.Var
		switch value := v.Value.(type) {
		case string:
			newVar.Value = ReplaceWithExtra(value, cache, extra)
		}
		newVar.Live = v.Live
		newVar.Sh = ReplaceWithExtra(v.Sh, cache, extra)
		newVar.Ref = v.Ref
		newVar.Json = ReplaceWithExtra(v.Json, cache, extra)
		newVar.Yaml = ReplaceWithExtra(v.Yaml, cache, extra)
		newVars.Set(k, newVar)
		return nil
	})

	return &newVars
}

// replaceValue is a recursive function that replaces all the string template
// variables in the given value with the values in the data map. If the value is
// a string, the function will replace the variables in the string and return
// the new string. If the value is a struct or a slice, the function will
// recursively call itself for each field or element of the struct or slice
// until all strings inside the struct or slice are replaced.
func replaceValue(copy, v reflect.Value, data map[string]any) error {
	switch v.Kind() {

	case reflect.Ptr:
		// Unwrap the pointer
		originalValue := v.Elem()
		// If the pointer is nil, do nothing
		if !originalValue.IsValid() {
			return nil
		}
		// Create an empty copy from the original value's type
		copy.Set(reflect.New(originalValue.Type()))
		// Unwrap the newly created pointer and call replaceValue recursively
		if err := replaceValue(copy.Elem(), originalValue, data); err != nil {
			return err
		}

	case reflect.Interface:
		// Unwrap the interface
		originalValue := v.Elem()
		// Create an empty copy from the original value's type
		copyValue := reflect.New(originalValue.Type())
		// Unwrap the newly created pointer and call replaceValue recursively
		if err := replaceValue(copyValue.Elem(), originalValue, data); err != nil {
			return err
		}
		copy.Set(copyValue)

	case reflect.Struct:
		// Loop over each field and call replaceValue recursively
		for i := 0; i < v.NumField(); i += 1 {
			if err := replaceValue(copy.Field(i), v.Field(i), data); err != nil {
				return err
			}
		}

	case reflect.Slice:
		// Create an empty copy from the original value's type
		copy.Set(reflect.MakeSlice(v.Type(), v.Len(), v.Cap()))
		// Loop over each element and call replaceValue recursively
		for i := 0; i < v.Len(); i += 1 {
			if err := replaceValue(copy.Index(i), v.Index(i), data); err != nil {
				return err
			}
		}

	case reflect.Map:
		// Create an empty copy from the original value's type
		copy.Set(reflect.MakeMap(v.Type()))
		// Loop over each key
		for _, key := range v.MapKeys() {
			// Create a copy of each map index
			originalValue := v.MapIndex(key)
			if originalValue.IsNil() {
				continue
			}
			copyValue := reflect.New(originalValue.Type()).Elem()
			// Call replaceValue recursively
			if err := replaceValue(copyValue, originalValue, data); err != nil {
				return err
			}
			copy.SetMapIndex(key, copyValue)
		}

	case reflect.String:
		tpl, err := template.New("").Funcs(templateFuncs).Parse(v.String())
		if err != nil {
			return err
		}
		var b bytes.Buffer
		if err := tpl.Execute(&b, data); err != nil {
			return err
		}
		str := strings.ReplaceAll(b.String(), "<no value>", "")
		copy.SetString(str)

	default:
		copy.Set(v)
	}

	return nil
}
