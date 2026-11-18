package templater

import (
	"bytes"
	"fmt"
	"maps"
	"regexp"
	"strings"

	"github.com/go-task/template"

	"github.com/go-task/task/v3/internal/deepcopy"
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

func ResolveRef(ref string, cache *Cache) any {
	// If there is already an error, do nothing
	if cache.err != nil {
		return nil
	}

	// Initialize the cache map if it's not already initialized
	if cache.cacheMap == nil {
		cache.cacheMap = cache.Vars.ToCacheMap()
	}

	if ref == "." {
		return cache.cacheMap
	}
	t, err := template.New("resolver").Funcs(templateFuncs).Parse(fmt.Sprintf("{{%s}}", ref))
	if err != nil {
		cache.err = err
		return nil
	}
	val, err := t.Resolve(cache.cacheMap)
	if err != nil {
		cache.err = err
		return nil
	}
	return val
}

func Replace[T any](v T, cache *Cache) T {
	return ReplaceWithExtra(v, cache, nil)
}

func ReplaceWithExtra[T any](v T, cache *Cache, extra map[string]any) T {
	// If there is already an error, do nothing
	if cache.err != nil {
		return v
	}

	// Optimization: skip if string is not a template
	if s, ok := any(v).(string); ok {
		if !strings.Contains(s, "{{") {
			return v
		}
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

	// Traverse the value and parse any template variables
	copy, err := deepcopy.TraverseStringsFunc(v, func(v string) (string, error) {
		// Optimization: skip if string is not a template
		if !strings.Contains(v, "{{") {
			return v, nil
		}
		tpl, err := template.New("").Funcs(templateFuncs).Parse(v)
		if err != nil {
			return v, err
		}
		var b bytes.Buffer
		if err := tpl.Execute(&b, data); err != nil {
			return v, err
		}
		return strings.ReplaceAll(b.String(), "<no value>", ""), nil
	})
	if err != nil {
		cache.err = err
		return v
	}

	return copy
}

// globVarPattern matches direct variable lookups in a glob pattern.
var globVarPattern = regexp.MustCompile(`\{\{\s*\.([a-zA-Z_][a-zA-Z0-9_.]*)\s*\}\}`)

// hasUnresolvedDynamicVar reports whether a glob references a shell-derived
// variable that has not been evaluated yet. FastGetVariables leaves such
// variables with an empty value so templates can still be parsed, which would
// turn "{{.DIR}}/**/*" into "/**/*" and make the globber scan from the
// filesystem root (and follow symlink cycles until it hangs).
func hasUnresolvedDynamicVar(glob string, cache *Cache) bool {
	for _, match := range globVarPattern.FindAllStringSubmatch(glob, -1) {
		v, ok := cache.Vars.Get(match[1])
		if !ok {
			return true
		}
		if v.Sh != nil && (v.Value == nil || v.Value == "") {
			return true
		}
	}
	return false
}

func ReplaceGlobs(globs []*ast.Glob, cache *Cache) []*ast.Glob {
	if cache.err != nil || len(globs) == 0 {
		return nil
	}

	new := make([]*ast.Glob, len(globs))
	for i, g := range globs {
		replaced := g.Glob
		if !hasUnresolvedDynamicVar(g.Glob, cache) {
			replaced = Replace(g.Glob, cache)
		}
		new[i] = &ast.Glob{
			Glob:   replaced,
			Negate: g.Negate,
		}
	}
	return new
}

func ReplaceVar(v ast.Var, cache *Cache) ast.Var {
	return ReplaceVarWithExtra(v, cache, nil)
}

func ReplaceVarWithExtra(v ast.Var, cache *Cache, extra map[string]any) ast.Var {
	if v.Ref != "" {
		return ast.Var{Value: ResolveRef(v.Ref, cache), Secret: v.Secret}
	}
	return ast.Var{
		Value:  ReplaceWithExtra(v.Value, cache, extra),
		Sh:     ReplaceWithExtra(v.Sh, cache, extra),
		Live:   v.Live,
		Ref:    v.Ref,
		Dir:    v.Dir,
		Secret: v.Secret,
	}
}

func ReplaceVars(vars *ast.Vars, cache *Cache) *ast.Vars {
	return ReplaceVarsWithExtra(vars, cache, nil)
}

func ReplaceVarsWithExtra(vars *ast.Vars, cache *Cache, extra map[string]any) *ast.Vars {
	if cache.err != nil || vars.Len() == 0 {
		return nil
	}

	newVars := ast.NewVars()
	for k, v := range vars.All() {
		newVars.Set(k, ReplaceVarWithExtra(v, cache, extra))
	}

	return newVars
}
