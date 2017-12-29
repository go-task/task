package task

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/go-task/task/internal/execext"

	"github.com/Masterminds/sprig"
	"github.com/mitchellh/go-homedir"
)

var (
	// TaskvarsFilePath file containing additional variables.
	TaskvarsFilePath = "Taskvars"
)

var (
	// ErrCantUnmarshalVar is returned for invalid var YAML.
	ErrCantUnmarshalVar = errors.New("task: can't unmarshal var value")
)

// Vars is a string[string] variables map.
type Vars map[string]Var

func getEnvironmentVariables() Vars {
	var (
		env = os.Environ()
		m   = make(Vars, len(env))
	)

	for _, e := range env {
		keyVal := strings.SplitN(e, "=", 2)
		key, val := keyVal[0], keyVal[1]
		m[key] = Var{Static: val}
	}
	return m
}

func (vs Vars) toStringMap() (m map[string]string) {
	m = make(map[string]string, len(vs))
	for k, v := range vs {
		if v.Sh != "" {
			// Dynamic variable is not yet resolved; trigger
			// <no value> to be used in templates.
			continue
		}
		m[k] = v.Static
	}
	return
}

// Var represents either a static or dynamic variable.
type Var struct {
	Static string
	Sh     string
}

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (v *Var) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err == nil {
		if strings.HasPrefix(str, "$") {
			v.Sh = strings.TrimPrefix(str, "$")
		} else {
			v.Static = str
		}
		return nil
	}

	var sh struct {
		Sh string
	}
	if err := unmarshal(&sh); err == nil {
		v.Sh = sh.Sh
		return nil
	}

	return ErrCantUnmarshalVar
}

// getVariables returns fully resolved variables following the priority order:
// 1. Call variables (should already have been resolved)
// 2. Environment (should not need to be resolved)
// 3. Task variables, resolved with access to:
//    - call, taskvars and environment variables
// 4. Taskvars variables, resolved with access to:
//    - environment variables
func (e *Executor) getVariables(call Call) (Vars, error) {
	t, ok := e.Taskfile.Tasks[call.Task]
	if !ok {
		return nil, &taskNotFoundError{call.Task}
	}

	merge := func(dest Vars, srcs ...Vars) {
		for _, src := range srcs {
			for k, v := range src {
				dest[k] = v
			}
		}
	}
	varsKeys := func(srcs ...Vars) []string {
		m := make(map[string]struct{})
		for _, src := range srcs {
			for k := range src {
				m[k] = struct{}{}
			}
		}
		lst := make([]string, 0, len(m))
		for k := range m {
			lst = append(lst, k)
		}
		return lst
	}
	replaceVars := func(dest Vars, keys []string) error {
		r := varReplacer{vars: dest}
		for _, k := range keys {
			v := dest[k]
			dest[k] = Var{
				Static: r.replace(v.Static),
				Sh:     r.replace(v.Sh),
			}
		}
		return r.err
	}
	resolveShell := func(dest Vars, keys []string) error {
		for _, k := range keys {
			v := dest[k]
			static, err := e.handleShVar(v)
			if err != nil {
				return err
			}
			dest[k] = Var{Static: static}
		}
		return nil
	}
	update := func(dest Vars, srcs ...Vars) error {
		merge(dest, srcs...)
		// updatedKeys ensures template evaluation is run only once.
		updatedKeys := varsKeys(srcs...)
		if err := replaceVars(dest, updatedKeys); err != nil {
			return err
		}
		return resolveShell(dest, updatedKeys)
	}

	// Resolve taskvars variables to "result" with environment override variables.
	override := getEnvironmentVariables()
	result := make(Vars, len(e.taskvars)+len(t.Vars)+len(override))
	if err := update(result, e.taskvars, override); err != nil {
		return nil, err
	}
	// Resolve task variables to "result" with environment and call override variables.
	merge(override, call.Vars)
	if err := update(result, t.Vars, override); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *Executor) handleShVar(v Var) (string, error) {
	if v.Static != "" || v.Sh == "" {
		return v.Static, nil
	}
	e.muDynamicCache.Lock()
	defer e.muDynamicCache.Unlock()

	if result, ok := e.dynamicCache[v.Sh]; ok {
		return result, nil
	}

	var stdout bytes.Buffer
	opts := &execext.RunCommandOptions{
		Command: v.Sh,
		Dir:     e.Dir,
		Stdout:  &stdout,
		Stderr:  e.Stderr,
	}
	if err := execext.RunCommand(opts); err != nil {
		return "", &dynamicVarError{cause: err, cmd: opts.Command}
	}

	// Trim a single trailing newline from the result to make most command
	// output easier to use in shell commands.
	result := strings.TrimSuffix(stdout.String(), "\n")

	e.dynamicCache[v.Sh] = result
	e.verboseErrf(`task: dynamic variable: '%s' result: '%s'`, v.Sh, result)

	return result, nil
}

// CompiledTask returns a copy of a task, but replacing variables in almost all
// properties using the Go template package.
func (e *Executor) CompiledTask(call Call) (*Task, error) {
	origTask, ok := e.Taskfile.Tasks[call.Task]
	if !ok {
		return nil, &taskNotFoundError{call.Task}
	}

	vars, err := e.getVariables(call)
	if err != nil {
		return nil, err
	}
	r := varReplacer{vars: vars}

	new := Task{
		Task:      origTask.Task,
		Desc:      r.replace(origTask.Desc),
		Sources:   r.replaceSlice(origTask.Sources),
		Generates: r.replaceSlice(origTask.Generates),
		Status:    r.replaceSlice(origTask.Status),
		Dir:       r.replace(origTask.Dir),
		Vars:      nil,
		Env:       r.replaceVars(origTask.Env),
		Silent:    origTask.Silent,
		Method:    r.replace(origTask.Method),
	}
	new.Dir, err = homedir.Expand(new.Dir)
	if err != nil {
		return nil, err
	}
	if e.Dir != "" && !filepath.IsAbs(new.Dir) {
		new.Dir = filepath.Join(e.Dir, new.Dir)
	}
	for k, v := range new.Env {
		static, err := e.handleShVar(v)
		if err != nil {
			return nil, err
		}
		new.Env[k] = Var{Static: static}
	}

	if len(origTask.Cmds) > 0 {
		new.Cmds = make([]*Cmd, len(origTask.Cmds))
		for i, cmd := range origTask.Cmds {
			new.Cmds[i] = &Cmd{
				Task:   r.replace(cmd.Task),
				Silent: cmd.Silent,
				Cmd:    r.replace(cmd.Cmd),
				Vars:   r.replaceVars(cmd.Vars),
			}

		}
	}
	if len(origTask.Deps) > 0 {
		new.Deps = make([]*Dep, len(origTask.Deps))
		for i, dep := range origTask.Deps {
			new.Deps[i] = &Dep{
				Task: r.replace(dep.Task),
				Vars: r.replaceVars(dep.Vars),
			}
		}
	}

	return &new, r.err
}

// varReplacer is a help struct that allow us to call "replaceX" funcs multiple
// times, without having to check for error each time. The first error that
// happen will be assigned to r.err, and consecutive calls to funcs will just
// return the zero value.
type varReplacer struct {
	vars   Vars
	strMap map[string]string
	err    error
}

func (r *varReplacer) replace(str string) string {
	if r.err != nil || str == "" {
		return ""
	}

	templ, err := template.New("").Funcs(templateFuncs).Parse(str)
	if err != nil {
		r.err = err
		return ""
	}

	if r.strMap == nil {
		r.strMap = r.vars.toStringMap()
	}

	var b bytes.Buffer
	if err = templ.Execute(&b, r.strMap); err != nil {
		r.err = err
		return ""
	}
	return b.String()
}

func (r *varReplacer) replaceSlice(strs []string) []string {
	if r.err != nil || len(strs) == 0 {
		return nil
	}

	new := make([]string, len(strs))
	for i, str := range strs {
		new[i] = r.replace(str)
	}
	return new
}

func (r *varReplacer) replaceVars(vars Vars) Vars {
	if r.err != nil || len(vars) == 0 {
		return nil
	}

	new := make(Vars, len(vars))
	for k, v := range vars {
		new[k] = Var{
			Static: r.replace(v.Static),
			Sh:     r.replace(v.Sh),
		}
	}
	return new
}

var (
	templateFuncs template.FuncMap
)

func init() {
	taskFuncs := template.FuncMap{
		"OS":   func() string { return runtime.GOOS },
		"ARCH": func() string { return runtime.GOARCH },
		"catLines": func(s string) string {
			s = strings.Replace(s, "\r\n", " ", -1)
			return strings.Replace(s, "\n", " ", -1)
		},
		"splitLines": func(s string) []string {
			s = strings.Replace(s, "\r\n", "\n", -1)
			return strings.Split(s, "\n")
		},
		"fromSlash": func(path string) string {
			return filepath.FromSlash(path)
		},
		"toSlash": func(path string) string {
			return filepath.ToSlash(path)
		},
		"exeExt": func() string {
			if runtime.GOOS == "windows" {
				return ".exe"
			}
			return ""
		},
		// IsSH is deprecated.
		"IsSH": func() bool { return true },
	}
	// Deprecated aliases for renamed functions.
	taskFuncs["FromSlash"] = taskFuncs["fromSlash"]
	taskFuncs["ToSlash"] = taskFuncs["toSlash"]
	taskFuncs["ExeExt"] = taskFuncs["exeExt"]

	templateFuncs = sprig.TxtFuncMap()
	for k, v := range taskFuncs {
		templateFuncs[k] = v
	}
}
