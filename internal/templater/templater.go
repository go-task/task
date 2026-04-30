package templater

import (
	"bytes"
	"fmt"
	"maps"
	"strings"
	"sync"

	"github.com/go-task/template"

	"github.com/go-task/task/v3/internal/deepcopy"
	"github.com/go-task/task/v3/taskfile/ast"
)

// dataMapPool reuses map[string]any backing memory across [ReplaceWithExtra]
// calls. When a template might mutate the dot via sprig's set/unset, we
// clone cache.cacheMap into a fresh map so those mutations cannot leak back
// into cache.cacheMap. The maps are short-lived and identically shaped, so a
// pool drops the per-call allocation without changing semantics.
var dataMapPool = sync.Pool{
	New: func() any {
		m := make(map[string]any, 64)
		return &m
	},
}

func acquireDataMap() *map[string]any {
	return dataMapPool.Get().(*map[string]any)
}

func releaseDataMap(m *map[string]any) {
	clear(*m)
	dataMapPool.Put(m)
}

// templateMayMutateDot reports whether a template source string might mutate
// its dot (root data map). The only functions registered in templateFuncs
// that mutate the dict passed to them are sprig's `set` and `unset`. Other
// sprig mutators (push/append/prepend on slices) operate on slice values
// retrieved from the dict — they do not modify the dict itself, and the
// original code's shallow [maps.Clone] never protected against those either.
//
// This is a conservative substring check: false positives are fine (they
// just force an unnecessary clone), false negatives would leak mutations.
// "set" appearing inside a literal string ({{"set"}}) or inside an unrelated
// identifier (settings, subset) is treated as potentially mutating, which is
// safe. Checking for "set" alone also catches "unset".
func templateMayMutateDot(s string) bool {
	return strings.Contains(s, "set")
}

// parsedTemplateCache memoizes parsed [template.Template] objects keyed by
// their source string. Each call to template.New("").Funcs(templateFuncs)
// copies the ~170-entry FuncMap into a fresh Template (sprig + go-task
// builtins) — that copy is the single largest allocator on the hot path of
// listing tasks in a project with many includes, because the same merged
// TaskfileEnv values are processed once per task.
//
// Caching the parsed Template is safe: once parsed, a Template may be
// executed concurrently from multiple goroutines (Go docs guarantee this),
// and Execute reads from but does not mutate the Template. The cache key is
// the unparsed source — sufficient because templateFuncs is a package-level
// singleton.
//
// We cache errors as well so that a malformed template surfaces the same
// error on every call instead of being re-parsed each time.
var parsedTemplateCache sync.Map // map[string]parsedTemplateEntry

type parsedTemplateEntry struct {
	tpl *template.Template
	err error
}

func cachedParse(text string) (*template.Template, error) {
	if v, ok := parsedTemplateCache.Load(text); ok {
		entry := v.(parsedTemplateEntry)
		return entry.tpl, entry.err
	}
	tpl, err := template.New("").Funcs(templateFuncs).Parse(text)
	actual, _ := parsedTemplateCache.LoadOrStore(text, parsedTemplateEntry{tpl: tpl, err: err})
	entry := actual.(parsedTemplateEntry)
	return entry.tpl, entry.err
}

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

// ResetErr clears the sticky error flag so a single Cache can be reused for
// a sequence of independent template operations. Without this, the first
// failed Replace would short-circuit every subsequent call on the same
// Cache.
func (r *Cache) ResetErr() {
	r.err = nil
}

// SyncVarSet updates the internal cacheMap after the caller mutates Vars via
// [ast.Vars.Set]. This lets a single Cache be reused across many Replace
// calls (each interleaved with a Vars.Set) without rebuilding the entire
// cacheMap from scratch via [ast.Vars.ToCacheMap]. The mirroring rules match
// [ast.Vars.ToCacheMap]: dynamic-only entries are excluded, Live takes
// precedence over Value.
//
// No-op if cacheMap has not yet been initialized — the lazy init in Replace
// will pick up the value on first use.
func (r *Cache) SyncVarSet(key string, v ast.Var) {
	if r.cacheMap == nil {
		return
	}
	if v.Sh != nil && *v.Sh != "" {
		delete(r.cacheMap, key)
		return
	}
	if v.Live != nil {
		r.cacheMap[key] = v.Live
	} else {
		r.cacheMap[key] = v.Value
	}
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
	t, err := cachedParse(fmt.Sprintf("{{%s}}", ref))
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

	// Initialize the cache map if it's not already initialized
	if cache.cacheMap == nil {
		cache.cacheMap = cache.Vars.ToCacheMap()
	}

	// Decide whether the template can read directly from cache.cacheMap or
	// whether we must clone it. Cloning is required when there is `extra`
	// data to merge in, or when any template traversed below might mutate
	// the dict via sprig's set/unset. For all other cases the clone is pure
	// overhead — tpl.Execute only reads from the data map.
	//
	// nil values (nil interface, nil *string) cannot run any template at
	// all, so they don't need a cloned data map. Strings are checked for
	// "set" anywhere. For other concrete types we conservatively clone.
	mayMutate := false
	switch {
	case extra != nil:
		mayMutate = true
	case any(v) == nil:
		// nil interface — TraverseStringsFunc has no strings to visit.
		mayMutate = false
	default:
		switch tv := any(v).(type) {
		case string:
			mayMutate = templateMayMutateDot(tv)
		case *string:
			// nil pointer — no template runs.
			mayMutate = tv != nil && templateMayMutateDot(*tv)
		default:
			// []string, map, struct, etc. — don't risk it.
			mayMutate = true
		}
	}

	var data map[string]any
	if mayMutate {
		dataPtr := acquireDataMap()
		defer releaseDataMap(dataPtr)
		data = *dataPtr
		maps.Copy(data, cache.cacheMap)
		if extra != nil {
			maps.Copy(data, extra)
		}
	} else {
		data = cache.cacheMap
	}

	// Traverse the value and parse any template variables
	copy, err := deepcopy.TraverseStringsFunc(v, func(v string) (string, error) {
		// Skip the template engine entirely if the string has no actions.
		// Go templates only fire on `{{`; pure literals can short-circuit.
		// This is a large win when iterating many merged taskfile env vars
		// whose values are static strings.
		if !strings.Contains(v, "{{") {
			return v, nil
		}
		tpl, err := cachedParse(v)
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

func ReplaceVar(v ast.Var, cache *Cache) ast.Var {
	return ReplaceVarWithExtra(v, cache, nil)
}

func ReplaceVarWithExtra(v ast.Var, cache *Cache, extra map[string]any) ast.Var {
	if v.Ref != "" {
		return ast.Var{Value: ResolveRef(v.Ref, cache)}
	}
	return ast.Var{
		Value: ReplaceWithExtra(v.Value, cache, extra),
		Sh:    ReplaceWithExtra(v.Sh, cache, extra),
		Live:  v.Live,
		Ref:   v.Ref,
		Dir:   v.Dir,
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
