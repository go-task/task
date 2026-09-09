package ast

import "github.com/go-task/task/v3/internal/deepcopy"

// DotenvScope preserves a declaration's context while Taskfiles are merged.
type DotenvScope struct {
	Namespace   string
	Location    string
	Files       []string
	Vars        *Vars
	Env         *Vars
	IncludeVars *Vars
}

func (d *DotenvScope) DeepCopy() *DotenvScope {
	if d == nil {
		return nil
	}
	return &DotenvScope{
		Namespace:   d.Namespace,
		Location:    d.Location,
		Files:       deepcopy.Slice(d.Files),
		Vars:        d.Vars.DeepCopy(),
		Env:         d.Env.DeepCopy(),
		IncludeVars: d.IncludeVars.DeepCopy(),
	}
}
