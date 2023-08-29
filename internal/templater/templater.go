package templater

import (
	"bytes"
	"maps"
	"strings"
	"text/template"

	"github.com/go-task/task/v3/taskfile/ast"
)

// Templater is a help struct that allow us to call "replaceX" funcs multiple
// times, without having to check for error each time. The first error that
// happen will be assigned to r.err, and consecutive calls to funcs will just
// return the zero value.
type Templater struct {
	Vars *ast.Vars

	cacheMap map[string]any
	err      error
}

func (r *Templater) ResetCache() {
	r.cacheMap = r.Vars.ToCacheMap()
}

func (r *Templater) Replace(str string) string {
	return r.replace(str, nil)
}

func (r *Templater) ReplaceWithExtra(str string, extra map[string]any) string {
	return r.replace(str, extra)
}

func (r *Templater) replace(str string, extra map[string]any) string {
	if r.err != nil || str == "" {
		return ""
	}

	templ, err := template.New("").Funcs(templateFuncs).Parse(str)
	if err != nil {
		r.err = err
		return ""
	}

	if r.cacheMap == nil {
		r.cacheMap = r.Vars.ToCacheMap()
	}

	var b bytes.Buffer
	if extra == nil {
		err = templ.Execute(&b, r.cacheMap)
	} else {
		// Copy the map to avoid modifying the cached map
		m := maps.Clone(r.cacheMap)
		maps.Copy(m, extra)
		err = templ.Execute(&b, m)
	}
	if err != nil {
		r.err = err
		return ""
	}
	return strings.ReplaceAll(b.String(), "<no value>", "")
}

func (r *Templater) ReplaceSlice(strs []string) []string {
	if r.err != nil || len(strs) == 0 {
		return nil
	}

	new := make([]string, len(strs))
	for i, str := range strs {
		new[i] = r.Replace(str)
	}
	return new
}

func (r *Templater) ReplaceGlobs(globs []*ast.Glob) []*ast.Glob {
	if r.err != nil || len(globs) == 0 {
		return nil
	}

	new := make([]*ast.Glob, len(globs))
	for i, g := range globs {
		new[i] = &ast.Glob{
			Glob:   r.Replace(g.Glob),
			Negate: g.Negate,
		}
	}
	return new
}

func (r *Templater) ReplaceVars(vars *ast.Vars) *ast.Vars {
	return r.replaceVars(vars, nil)
}

func (r *Templater) ReplaceVarsWithExtra(vars *ast.Vars, extra map[string]any) *ast.Vars {
	return r.replaceVars(vars, extra)
}

func (r *Templater) replaceVars(vars *ast.Vars, extra map[string]any) *ast.Vars {
	if r.err != nil || vars.Len() == 0 {
		return nil
	}

	var newVars ast.Vars
	_ = vars.Range(func(k string, v ast.Var) error {
		var newVar ast.Var
		switch value := v.Value.(type) {
		case string:
			newVar.Value = r.ReplaceWithExtra(value, extra)
		}
		newVar.Live = v.Live
		newVar.Sh = r.ReplaceWithExtra(v.Sh, extra)
		newVar.Ref = v.Ref
		newVar.Json = r.ReplaceWithExtra(v.Json, extra)
		newVar.Yaml = r.ReplaceWithExtra(v.Yaml, extra)
		newVars.Set(k, newVar)
		return nil
	})

	return &newVars
}

func (r *Templater) Err() error {
	return r.err
}
