package templater

import (
	"github.com/go-task/task/v3/taskfile/ast"
)

// MaskSecrets replaces template placeholders with their values, masking secrets.
// This function uses the Go templater to resolve all variables ({{.VAR}}) while
// masking secret ones as "*****".
func MaskSecrets(cmdTemplate string, vars *ast.Vars) string {
	if vars == nil || vars.Len() == 0 {
		return cmdTemplate
	}

	// Create a cache map with secrets masked
	maskedVars := vars.DeepCopy()
	for name, v := range maskedVars.All() {
		if v.Secret {
			// Replace secret value with mask
			maskedVars.Set(name, ast.Var{
				Value:  "*****",
				Secret: true,
			})
		}
	}

	// Use the templater to resolve the template with masked secrets
	cache := &Cache{Vars: maskedVars}
	result := Replace(cmdTemplate, cache)

	// If there was an error, return the original template
	if cache.Err() != nil {
		return cmdTemplate
	}

	return result
}
