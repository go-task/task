package templater

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"mvdan.cc/sh/v3/shell"
	"mvdan.cc/sh/v3/syntax"

	sprig "github.com/go-task/slim-sprig/v3"
	"github.com/go-task/template"
)

var templateFuncs template.FuncMap

func init() {
	taskFuncs := template.FuncMap{
		"OS":   func() string { return runtime.GOOS },
		"ARCH": func() string { return runtime.GOARCH },
		"catLines": func(s string) string {
			s = strings.ReplaceAll(s, "\r\n", " ")
			return strings.ReplaceAll(s, "\n", " ")
		},
		"splitLines": func(s string) []string {
			s = strings.ReplaceAll(s, "\r\n", "\n")
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
		"shellQuote": func(str string) (string, error) {
			return syntax.Quote(str, syntax.LangBash)
		},
		"splitArgs": func(s string) ([]string, error) {
			return shell.Fields(s, nil)
		},
		// IsSH is deprecated.
		"IsSH": func() bool { return true },
		"joinPath": func(elem ...string) string {
			return filepath.Join(elem...)
		},
		"relPath": func(basePath, targetPath string) (string, error) {
			return filepath.Rel(basePath, targetPath)
		},
		"merge": func(base map[string]any, v ...map[string]any) map[string]any {
			cap := len(v)
			for _, m := range v {
				cap += len(m)
			}
			result := make(map[string]any, cap)
			for k, v := range base {
				result[k] = v
			}
			for _, m := range v {
				for k, v := range m {
					result[k] = v
				}
			}
			return result
		},
		"spew": func(v any) string {
			return spew.Sdump(v)
		},
	}

	// aliases
	taskFuncs["q"] = taskFuncs["shellQuote"]

	// Deprecated aliases for renamed functions.
	taskFuncs["FromSlash"] = taskFuncs["fromSlash"]
	taskFuncs["ToSlash"] = taskFuncs["toSlash"]
	taskFuncs["ExeExt"] = taskFuncs["exeExt"]

	templateFuncs = template.FuncMap(sprig.TxtFuncMap())
	for k, v := range taskFuncs {
		templateFuncs[k] = v
	}
}
