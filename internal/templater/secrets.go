package templater

import (
	"github.com/go-task/task/v3/taskfile/ast"
)

// MaskSecrets replaces template placeholders with their values, masking secrets.
// This function uses the Go templater to resolve all variables ({{.VAR}}) while
// masking secret ones as "*****".
func MaskSecrets(cmdTemplate string, vars *ast.Vars) string {
	return MaskSecretsWithExtra(cmdTemplate, vars, nil)
}

// MaskSecretsWithExtra is like MaskSecrets but also resolves extra variables (e.g., loop vars).
func MaskSecretsWithExtra(cmdTemplate string, vars *ast.Vars, extra map[string]any) string {
	if vars == nil {
		vars = ast.NewVars()
	}

	// Fast path: if there are no secrets to mask, resolve the template directly
	// without the extra DeepCopy + masking pass.
	if !hasSecrets(vars) {
		cache := &Cache{Vars: vars}
		result := ReplaceWithExtra(cmdTemplate, cache, extra)
		if cache.Err() != nil {
			return cmdTemplate
		}
		return result
	}

	// Create a copy with secret values masked, leaving the originals untouched.
	maskedVars := vars.DeepCopy()
	for name, v := range maskedVars.All() {
		if v.Secret {
			maskedVars.Set(name, ast.Var{
				Value:  "*****",
				Secret: true,
			})
		}
	}

	cache := &Cache{Vars: maskedVars}
	result := ReplaceWithExtra(cmdTemplate, cache, extra)

	// If there was an error, return the original template
	if cache.Err() != nil {
		return cmdTemplate
	}

	return result
}

// hasSecrets reports whether any variable is marked as secret.
func hasSecrets(vars *ast.Vars) bool {
	for _, v := range vars.All() {
		if v.Secret {
			return true
		}
	}
	return false
}
