package taskfuncs

import (
	"math/rand/v2"
	"path/filepath"
	"runtime"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-sprout/sprout"
	"github.com/google/uuid"
)

// Registry exposes Task's own template functions to a sprout handler.
type Registry struct {
	handler sprout.Handler
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) UID() string {
	return "go-task/task.taskfuncs"
}

func (r *Registry) LinkHandler(fh sprout.Handler) error {
	r.handler = fh
	return nil
}

func (r *Registry) RegisterFunctions(fnMap sprout.FunctionMap) error {
	for name, fn := range functions() {
		sprout.AddFunction(fnMap, name, fn)
	}
	return nil
}

func (r *Registry) RegisterAliases(aliasMap sprout.FunctionAliasMap) error {
	sprout.AddAlias(aliasMap, "shellQuote", "q")
	// Deprecated aliases for renamed functions.
	sprout.AddAlias(aliasMap, "fromSlash", "FromSlash")
	sprout.AddAlias(aliasMap, "toSlash", "ToSlash")
	sprout.AddAlias(aliasMap, "exeExt", "ExeExt")
	return nil
}

func (r *Registry) RegisterNotices(notices *[]sprout.FunctionNotice) error {
	sprout.AddNotice(notices, sprout.NewDeprecatedNotice("IsSH", "it always returns true and can be removed from your templates"))
	sprout.AddNotice(notices, sprout.NewDeprecatedNotice("FromSlash", "please use `fromSlash` instead"))
	sprout.AddNotice(notices, sprout.NewDeprecatedNotice("ToSlash", "please use `toSlash` instead"))
	sprout.AddNotice(notices, sprout.NewDeprecatedNotice("ExeExt", "please use `exeExt` instead"))
	return nil
}

// Overrides returns the functions that must be re-applied after the handler is
// built. sprout owns `merge`, and its encoding registry aliases `fromYAML`
// and `toYAML` onto the camelCase names Task already uses — and AssignAliases
// overwrites unconditionally, so registration order alone cannot protect them.
// Their semantics differ from Task's (deep merge, and errors raised instead of
// swallowed), hence Task's implementations win.
func Overrides() sprout.FunctionMap {
	return sprout.FunctionMap{
		"merge":        Merge,
		"fromYaml":     FromYAML,
		"mustFromYaml": MustFromYAML,
		"toYaml":       ToYAML,
		"mustToYaml":   MustToYAML,
	}
}

func functions() sprout.FunctionMap {
	return sprout.FunctionMap{
		"OS":           OS,
		"ARCH":         Arch,
		"numCPU":       runtime.NumCPU,
		"catLines":     CatLines,
		"splitLines":   SplitLines,
		"fromSlash":    filepath.FromSlash,
		"toSlash":      filepath.ToSlash,
		"exeExt":       ExeExt,
		"shellQuote":   ShellQuote,
		"splitArgs":    SplitArgs,
		"IsSH":         IsSH, // Deprecated
		"joinPath":     filepath.Join,
		"joinEnv":      JoinEnv,
		"joinUrl":      JoinURL,
		"relPath":      filepath.Rel,
		"absPath":      filepath.Abs,
		"merge":        Merge,
		"spew":         spew.Sdump,
		"fromYaml":     FromYAML,
		"mustFromYaml": MustFromYAML,
		"toYaml":       ToYAML,
		"mustToYaml":   MustToYAML,
		"uuid":         uuid.New,
		"randIntN":     rand.IntN,
	}
}
