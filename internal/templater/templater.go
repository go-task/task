package templater

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/go-task/task/v3/taskfile"
)

// Templater is a help struct that allow us to call "replaceX" funcs multiple
// times, without having to check for error each time. The first error that
// happen will be assigned to r.err, and consecutive calls to funcs will just
// return the zero value.
type Templater struct {
	Vars          *taskfile.Vars
	RemoveNoValue bool

	cacheMap map[string]any
	err      error
}

func (r *Templater) ResetCache() {
	r.cacheMap = r.Vars.ToCacheMap()
}

func (r *Templater) Replace(str string) string {
	if r.err != nil || str == "" {
		return ""
	}

	templ, err := template.New("").Funcs(TemplateFuncs).Parse(str)
	if err != nil {
		r.err = err
		return ""
	}

	if r.cacheMap == nil {
		r.cacheMap = r.Vars.ToCacheMap()
	}

	var b bytes.Buffer
	if err = templ.Execute(&b, r.cacheMap); err != nil {
		r.err = err
		return ""
	}
	if r.RemoveNoValue {
		return strings.ReplaceAll(b.String(), "<no value>", "")
	}
	return b.String()
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

func (r *Templater) ReplaceVars(vars *taskfile.Vars) *taskfile.Vars {
	if r.err != nil || vars.Len() == 0 {
		return nil
	}

	var new taskfile.Vars
	_ = vars.Range(func(k string, v taskfile.Var) error {
		new.Set(k, taskfile.Var{
			Static: r.Replace(v.Static),
			Live:   v.Live,
			Sh:     r.Replace(v.Sh),
		})
		return nil
	})

	return &new
}

func (r *Templater) Err() error {
	return r.err
}
