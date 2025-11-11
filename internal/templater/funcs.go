package templater

import (
	"maps"
	"math/rand/v2"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"go.yaml.in/yaml/v4"
	"mvdan.cc/sh/v3/shell"
	"mvdan.cc/sh/v3/syntax"

	sprig "github.com/go-task/slim-sprig/v3"
	"github.com/go-task/template"
)

var templateFuncs template.FuncMap

func init() {
	taskFuncs := template.FuncMap{
		"OS":           os,
		"ARCH":         arch,
		"numCPU":       runtime.NumCPU,
		"catLines":     catLines,
		"splitLines":   splitLines,
		"fromSlash":    filepath.FromSlash,
		"toSlash":      filepath.ToSlash,
		"exeExt":       exeExt,
		"shellQuote":   shellQuote,
		"splitArgs":    splitArgs,
		"IsSH":         IsSH, // Deprecated
		"joinPath":     filepath.Join,
		"relPath":      filepath.Rel,
		"merge":        merge,
		"spew":         spew.Sdump,
		"fromYaml":     fromYaml,
		"mustFromYaml": mustFromYaml,
		"toYaml":       toYaml,
		"mustToYaml":   mustToYaml,
		"uuid":         uuid.New,
		"randIntN":     rand.IntN,
	}

	// aliases
	taskFuncs["q"] = taskFuncs["shellQuote"]

	// Deprecated aliases for renamed functions.
	taskFuncs["FromSlash"] = taskFuncs["fromSlash"]
	taskFuncs["ToSlash"] = taskFuncs["toSlash"]
	taskFuncs["ExeExt"] = taskFuncs["exeExt"]

	templateFuncs = template.FuncMap(sprig.TxtFuncMap())
	maps.Copy(templateFuncs, taskFuncs)
}

func os() string {
	return runtime.GOOS
}

func arch() string {
	return runtime.GOARCH
}

func catLines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	return strings.ReplaceAll(s, "\n", " ")
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}

func exeExt() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func shellQuote(str string) (string, error) {
	return syntax.Quote(str, syntax.LangBash)
}

func splitArgs(s string) ([]string, error) {
	return shell.Fields(s, nil)
}

// Deprecated: now always returns true
func IsSH() bool {
	return true
}

func merge(base map[string]any, v ...map[string]any) map[string]any {
	cap := len(v)
	for _, m := range v {
		cap += len(m)
	}
	result := make(map[string]any, cap)
	maps.Copy(result, base)
	for _, m := range v {
		maps.Copy(result, m)
	}
	return result
}

func fromYaml(v string) any {
	output, _ := mustFromYaml(v)
	return output
}

func mustFromYaml(v string) (any, error) {
	var output any
	err := yaml.Unmarshal([]byte(v), &output)
	return output, err
}

func toYaml(v any) string {
	output, _ := yaml.Marshal(v)
	return string(output)
}

func mustToYaml(v any) (string, error) {
	output, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(output), nil
}
