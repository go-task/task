package templater

import (
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	sprig "github.com/go-task/slim-sprig"
	"mvdan.cc/sh/v3/syntax"
)

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
		"shellQuote": func(str string) (string, error) {
			return syntax.Quote(str, syntax.LangBash)
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
