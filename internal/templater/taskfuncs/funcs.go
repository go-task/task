// Package taskfuncs provides the template functions that Task adds on top of
// the generic ones supplied by sprout.
package taskfuncs

import (
	"maps"
	"net/url"
	"os"
	"runtime"
	"strings"

	"go.yaml.in/yaml/v3"
	"mvdan.cc/sh/v3/shell"
	"mvdan.cc/sh/v3/syntax"
)

func OS() string {
	return runtime.GOOS
}

func Arch() string {
	return runtime.GOARCH
}

func CatLines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	return strings.ReplaceAll(s, "\n", " ")
}

func SplitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}

func ExeExt() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func ShellQuote(str string) (string, error) {
	return syntax.Quote(str, syntax.LangBash)
}

func SplitArgs(s string) ([]string, error) {
	return shell.Fields(s, nil)
}

// Deprecated: now always returns true
func IsSH() bool {
	return true
}

func JoinEnv(elem ...string) string {
	return strings.Join(elem, string(os.PathListSeparator))
}

func JoinURL(elem ...string) (string, error) {
	if len(elem) == 0 {
		return "", nil
	}
	// Use net/url.JoinPath rather than path.Join: the latter runs path.Clean,
	// which collapses the "//" in a URL scheme (e.g. "http://" -> "http:/").
	return url.JoinPath(elem[0], elem[1:]...)
}

// Merge shallow-merges maps, later keys winning. It shadows sprout's `merge`,
// which deep-merges and keeps the destination value on conflict.
func Merge(base map[string]any, v ...map[string]any) map[string]any {
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

func FromYAML(v string) any {
	output, _ := MustFromYAML(v)
	return output
}

func MustFromYAML(v string) (any, error) {
	var output any
	err := yaml.Unmarshal([]byte(v), &output)
	return output, err
}

func ToYAML(v any) string {
	output, _ := yaml.Marshal(v)
	return string(output)
}

func MustToYAML(v any) (string, error) {
	output, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(output), nil
}
