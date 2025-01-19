package taskfunc

import (
	"github.com/go-sprout/sprout"
)

// GoTaskRegistry struct implements the Registry interface,
// providing a way to register functions and aliases for go-task templates.
type GoTaskRegistry struct {
	handler sprout.Handler // Embedding Handler for shared functionality
}

// NewRegistry initializes and returns a new instance of your registry.
func NewRegistry() *GoTaskRegistry {
	return &GoTaskRegistry{}
}

// UID provides a unique identifier for your registry.
func (gtr *GoTaskRegistry) UID() string {
	return "go-task/task.gotaskregistry"
}

// LinkHandler connects the Handler to your registry, enabling runtime functionalities.
func (gtr *GoTaskRegistry) LinkHandler(fh sprout.Handler) error {
	gtr.handler = fh
	return nil
}

// RegisterFunctions adds the provided functions into the given function map.
// This method is called by an Handler to register all functions of a registry.
func (gtr *GoTaskRegistry) RegisterFunctions(funcsMap sprout.FunctionMap) error {
	sprout.AddFunction(funcsMap, "os", gtr.OS)
	sprout.AddFunction(funcsMap, "arch", gtr.ARCH)
	sprout.AddFunction(funcsMap, "numCPU", gtr.NumCPU)
	sprout.AddFunction(funcsMap, "catLines", gtr.CatLines)
	sprout.AddFunction(funcsMap, "splitLines", gtr.SplitLines)
	sprout.AddFunction(funcsMap, "fromSlash", gtr.FromSlash)
	sprout.AddFunction(funcsMap, "toSlash", gtr.ToSlash)
	sprout.AddFunction(funcsMap, "exeExt", gtr.ExeExt)
	sprout.AddFunction(funcsMap, "shellQuote", gtr.ShellQuote)
	sprout.AddFunction(funcsMap, "splitArgs", gtr.SplitArgs)
	sprout.AddFunction(funcsMap, "IsSH", gtr.IsSH)
	sprout.AddFunction(funcsMap, "joinPath", gtr.JoinPath)
	sprout.AddFunction(funcsMap, "relPath", gtr.RelPath)
	// sprout.AddFunction(funcsMap, "merge", gtr.Merge) // can be replaced by maps.mergeOverwrite
	sprout.AddFunction(funcsMap, "spew", gtr.Spew)

	return nil
}

// RegisterAliases adds the provided aliases into the given alias map.
// method is called by an Handler to register all aliases of a registry.
func (gtr *GoTaskRegistry) RegisterAliases(aliasMap sprout.FunctionAliasMap) error {
	sprout.AddAlias(aliasMap, "shellQuote", "q")
	// Overwriting aliases
	sprout.AddAlias(aliasMap, "mergeOverwrite", "merge")
	// Deprecated aliases
	sprout.AddAlias(aliasMap, "fromSlash", "FromSlash")
	sprout.AddAlias(aliasMap, "toSlash", "ToSlash")
	sprout.AddAlias(aliasMap, "exeExt", "ExeExt")
	sprout.AddAlias(aliasMap, "os", "OS")
	sprout.AddAlias(aliasMap, "arch", "ARCH")
	return nil
}

func (gtr *GoTaskRegistry) RegisterNotices(notices *[]sprout.FunctionNotice) error {
	sprout.AddNotice(notices, sprout.NewDeprecatedNotice("IsSH", "This function is deprecated. Consider removing it from your templates."))
	sprout.AddNotice(notices, sprout.NewDeprecatedNotice("FromSlash", "Use `fromSlash` instead."))
	sprout.AddNotice(notices, sprout.NewDeprecatedNotice("ToSlash", "Use `toSlash` instead."))
	sprout.AddNotice(notices, sprout.NewDeprecatedNotice("ExeExt", "Use `exeExt` instead."))
	sprout.AddNotice(notices, sprout.NewDeprecatedNotice("OS", "Use `os` instead."))
	sprout.AddNotice(notices, sprout.NewDeprecatedNotice("ARCH", "Use `arch` instead."))
	return nil
}
