package taskfunc

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"mvdan.cc/sh/v3/shell"
	"mvdan.cc/sh/v3/syntax"
)

// OS returns the running program's operating system target: one of
// "darwin", "dragonfly", "freebsd", "linux", "netbsd", "openbsd", "plan9",
// "solaris", or "windows".
//
// Returns:
//
//	string - the operating system target of the running program.
func (*GoTaskRegistry) OS() string {
	return runtime.GOOS
}

// ARCH returns the running program's architecture target: one of "386", "amd64",
// "arm", "arm64", "ppc64", "ppc64le", "mips", "mipsle", "mips64", "mips64le",
// "s390", "s390x".
//
// Returns:
//
//	string - the architecture target of the running program.
func (*GoTaskRegistry) ARCH() string {
	return runtime.GOARCH
}

// NumCPU returns the number of logical CPUs usable by the current process.
//
// Returns:
//
//	int - the number of logical CPUs.
func (*GoTaskRegistry) NumCPU() int {
	return runtime.NumCPU()
}

// CatLines replaces Unix (`\n`) and Windows (`\r\n`) styled newlines with a space.
// This is useful to ensure the string is a single line regardless of the newline style.
//
// Parameters:
//
//	s - the string to replace newlines in.
//
// Returns:
//
//	string - the string with newlines replaced by spaces.
func (*GoTaskRegistry) CatLines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	return strings.ReplaceAll(s, "\n", " ")
}

// SplitLines splits Unix (`\n`) and Windows (`\r\n`) styled newlines.
// This is useful to split a string into lines regardless of the newline style.
//
// Parameters:
//
//	s - the string to split into lines.
//
// Returns:
//
//	[]string - the string split into lines.
func (*GoTaskRegistry) SplitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}

// FromSlash converts a string from a slash (`/`) path format to a platform-specific
// path format. Does nothing on Unix, but on Windows converts a string from a slash
// path format to a backslash path format.
//
// Parameters:
//
//	path - the string to convert.
//
// Returns:
//
//	string - the path in a platform-specific format.
func (*GoTaskRegistry) FromSlash(path string) string {
	return filepath.FromSlash(path)
}

// ToSlash converts a string from a platform-specific path format to a slash
// path format. Does nothing on Unix, but on Windows converts a string from a
// backslash path format to a slash path format.
//
// Parameters:
//
//	path - the string to convert.
//
// Returns:
//
//	string - the path in a slash path format.
func (*GoTaskRegistry) ToSlash(path string) string {
	return filepath.ToSlash(path)
}

// ExeExt returns the appropriate executable file extension for the current
// operating system. On Windows, it returns ".exe", while on other operating
// systems, it returns an empty string.
//
// Returns:
//
//	string - the executable file extension for the current OS.

func (*GoTaskRegistry) ExeExt() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// ShellQuote quotes a string to make it safe for use in shell scripts.
// It assumes the Bash dialect.
//
// Parameters:
//
//	str - the string to quote.
//
// Returns:
//
//	string - the quoted string.
//	error - an error if unable to quote the string.
func (*GoTaskRegistry) ShellQuote(str string) (string, error) {
	return syntax.Quote(str, syntax.LangBash)
}

// SplitArgs splits a string as if it were a command's arguments. Task uses
// [this Go function](https://pkg.go.dev/mvdan.cc/sh/v3@v3.4.0/shell#Fields).
//
// Parameters:
//
//	s - the string to split into arguments.
//
// Returns:
//
//	[]string - the string split into arguments.
//	error - an error if unable to split the string.
func (*GoTaskRegistry) SplitArgs(s string) ([]string, error) {
	return shell.Fields(s, nil)
}

// ! IsSH is deprecated.
func (*GoTaskRegistry) IsSH() bool {
	return true
}

// JoinPath joins any number of arguments into a path. The same as Go's
// [filepath.Join](https://pkg.go.dev/path/filepath#Join).
//
// Parameters:
//
//	elem - the elements to join together.
//
// Returns:
//
//	string - the joined path.
func (*GoTaskRegistry) JoinPath(elem ...string) string {
	return filepath.Join(elem...)
}

// RelPath converts an absolute target path into a relative path, based on a base path.
// This function utilizes Go's filepath.Rel to perform the conversion.
//
// Parameters:
//
//	basePath - the base path from which the relative path is calculated.
//	targetPath - the absolute path to be converted into a relative path.
//
// Returns:
//
//	string - the relative path from basePath to targetPath.
//	error - an error if the paths cannot be made relative.

func (*GoTaskRegistry) RelPath(basePath, targetPath string) (string, error) {
	return filepath.Rel(basePath, targetPath)
}

// Merge creates a new map that is a copy of the first map with the keys of each
// subsequent map merged into it. If there is a duplicate key, the value of the
// last map with that key is used.
//
// Parameters:
//
//	base - the base map to merge subsequent maps into.
//	v - the maps to merge into the base map.
//
// Returns:
//
//	map[string]any - the merged map.
func (*GoTaskRegistry) Merge(base map[string]any, v ...map[string]any) map[string]any {
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
}

// Spew returns the Go representation of a specific variable. Useful for
// debugging. Uses the [davecgh/go-spew](https://github.com/davecgh/go-spew)
// package.
//
// Parameters:
//
//	v - the variable to dump.
//
// Returns:
//
//	string - the dumped variable.
func (*GoTaskRegistry) Spew(v any) string {
	return spew.Sdump(v)
}
