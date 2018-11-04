package templater

import (
	"bytes"
	"text/template"

	"github.com/go-task/task/v2/internal/taskfile"
)

// Templater is a help struct that allow us to call "replaceX" funcs multiple
// times, without having to check for error each time. The first error that
// happen will be assigned to r.err, and consecutive calls to funcs will just
// return the zero value.
type Templater struct {
	Vars taskfile.Vars

	strMap map[string]string
	err    error
}

func (r *Templater) Replace(str string) string {
	if r.err != nil || str == "" {
		return ""
	}

	templ, err := template.New("").Funcs(templateFuncs).Parse(str)
	if err != nil {
		r.err = err
		return ""
	}

	if r.strMap == nil {
		r.strMap = r.Vars.ToStringMap()
	}

	var b bytes.Buffer
	if err = templ.Execute(&b, r.strMap); err != nil {
		r.err = err
		return ""
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

func (r *Templater) ReplaceVars(vars taskfile.Vars) taskfile.Vars {
	if r.err != nil || len(vars) == 0 {
		return nil
	}

	new := make(taskfile.Vars, len(vars))
	for k, v := range vars {
		new[k] = taskfile.Var{
			Static: r.Replace(v.Static),
			Sh:     r.Replace(v.Sh),
		}
	}
	return new
}

func (r *Templater) Err() error {
	return r.err
}
