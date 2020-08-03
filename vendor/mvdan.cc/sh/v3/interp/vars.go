// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package interp

import (
	"os"
	"runtime"
	"strconv"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/syntax"
)

type overlayEnviron struct {
	parent expand.Environ
	values map[string]expand.Variable
}

func (o overlayEnviron) Get(name string) expand.Variable {
	if vr, ok := o.values[name]; ok {
		return vr
	}
	return o.parent.Get(name)
}

func (o overlayEnviron) Set(name string, vr expand.Variable) {
	o.values[name] = vr
}

func (o overlayEnviron) Each(f func(name string, vr expand.Variable) bool) {
	o.parent.Each(f)
	for name, vr := range o.values {
		if !f(name, vr) {
			return
		}
	}
}

func execEnv(env expand.Environ) []string {
	list := make([]string, 0, 64)
	env.Each(func(name string, vr expand.Variable) bool {
		if !vr.IsSet() {
			// If a variable is set globally but unset in the
			// runner, we need to ensure it's not part of the final
			// list. Seems like zeroing the element is enough.
			// This is a linear search, but this scenario should be
			// rare, and the number of variables shouldn't be large.
			for i, kv := range list {
				if strings.HasPrefix(kv, name+"=") {
					list[i] = ""
				}
			}
		}
		if vr.Exported {
			list = append(list, name+"="+vr.String())
		}
		return true
	})
	return list
}

func (r *Runner) lookupVar(name string) expand.Variable {
	if name == "" {
		panic("variable name must not be empty")
	}
	var vr expand.Variable
	switch name {
	case "#":
		vr.Kind, vr.Str = expand.String, strconv.Itoa(len(r.Params))
	case "@", "*":
		vr.Kind, vr.List = expand.Indexed, r.Params
	case "?":
		vr.Kind, vr.Str = expand.String, strconv.Itoa(r.lastExit)
	case "$":
		vr.Kind, vr.Str = expand.String, strconv.Itoa(os.Getpid())
	case "PPID":
		vr.Kind, vr.Str = expand.String, strconv.Itoa(os.Getppid())
	case "DIRSTACK":
		vr.Kind, vr.List = expand.Indexed, r.dirStack
	case "0":
		vr.Kind = expand.String
		if r.filename != "" {
			vr.Str = r.filename
		} else {
			vr.Str = "gosh"
		}
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		vr.Kind = expand.String
		i := int(name[0] - '1')
		if i < len(r.Params) {
			vr.Str = r.Params[i]
		} else {
			vr.Str = ""
		}
	}
	if vr.IsSet() {
		return vr
	}
	if value, e := r.cmdVars[name]; e {
		return expand.Variable{Kind: expand.String, Str: value}
	}
	if vr, e := r.funcVars[name]; e {
		vr.Local = true
		return vr
	}
	if vr, e := r.Vars[name]; e {
		return vr
	}
	if vr := r.Env.Get(name); vr.IsSet() {
		return vr
	}
	if runtime.GOOS == "windows" {
		upper := strings.ToUpper(name)
		if vr := r.Env.Get(upper); vr.IsSet() {
			return vr
		}
	}
	if r.opts[optNoUnset] {
		r.errf("%s: unbound variable\n", name)
		r.exit = 1
		r.exitShell = true
	}
	return expand.Variable{}
}

func (r *Runner) envGet(name string) string {
	return r.lookupVar(name).String()
}

func (r *Runner) delVar(name string) {
	vr := r.lookupVar(name)
	if vr.ReadOnly {
		r.errf("%s: readonly variable\n", name)
		r.exit = 1
		return
	}
	if vr.Local {
		// don't overwrite a non-local var with the same name
		r.funcVars[name] = expand.Variable{}
	} else {
		r.Vars[name] = expand.Variable{} // to not query r.Env
	}
}

func (r *Runner) setVarString(name, value string) {
	r.setVar(name, nil, expand.Variable{Kind: expand.String, Str: value})
}

func (r *Runner) setVarInternal(name string, vr expand.Variable) {
	if vr.Kind == expand.String {
		if r.opts[optAllExport] {
			vr.Exported = true
		}
	} else {
		vr.Exported = false
	}
	if vr.Local {
		if r.funcVars == nil {
			r.funcVars = make(map[string]expand.Variable)
		}
		r.funcVars[name] = vr
	} else {
		r.Vars[name] = vr
	}
}

func (r *Runner) setVar(name string, index syntax.ArithmExpr, vr expand.Variable) {
	cur := r.lookupVar(name)
	if cur.ReadOnly {
		r.errf("%s: readonly variable\n", name)
		r.exit = 1
		return
	}
	if name2, var2 := cur.Resolve(r.Env); name2 != "" {
		name = name2
		cur = var2
	}

	if vr.Kind == expand.String && index == nil {
		// When assigning a string to an array, fall back to the
		// zero value for the index.
		switch cur.Kind {
		case expand.Indexed:
			index = &syntax.Word{Parts: []syntax.WordPart{
				&syntax.Lit{Value: "0"},
			}}
		case expand.Associative:
			index = &syntax.Word{Parts: []syntax.WordPart{
				&syntax.DblQuoted{},
			}}
		}
	}
	if index == nil {
		r.setVarInternal(name, vr)
		return
	}

	// from the syntax package, we know that value must be a string if index
	// is non-nil; nested arrays are forbidden.
	valStr := vr.Str

	var list []string
	switch cur.Kind {
	case expand.String:
		list = append(list, cur.Str)
	case expand.Indexed:
		list = cur.List
	case expand.Associative:
		// if the existing variable is already an AssocArray, try our
		// best to convert the key to a string
		w, ok := index.(*syntax.Word)
		if !ok {
			return
		}
		k := r.literal(w)
		cur.Map[k] = valStr
		r.setVarInternal(name, cur)
		return
	}
	k := r.arithm(index)
	for len(list) < k+1 {
		list = append(list, "")
	}
	list[k] = valStr
	cur.Kind = expand.Indexed
	cur.List = list
	r.setVarInternal(name, cur)
}

func (r *Runner) setFunc(name string, body *syntax.Stmt) {
	if r.Funcs == nil {
		r.Funcs = make(map[string]*syntax.Stmt, 4)
	}
	r.Funcs[name] = body
}

func stringIndex(index syntax.ArithmExpr) bool {
	w, ok := index.(*syntax.Word)
	if !ok || len(w.Parts) != 1 {
		return false
	}
	switch w.Parts[0].(type) {
	case *syntax.DblQuoted, *syntax.SglQuoted:
		return true
	}
	return false
}

func (r *Runner) assignVal(as *syntax.Assign, valType string) expand.Variable {
	prev := r.lookupVar(as.Name.Value)
	if as.Naked {
		return prev
	}
	if as.Value != nil {
		s := r.literal(as.Value)
		if !as.Append || !prev.IsSet() {
			prev.Kind = expand.String
			if valType == "-n" {
				prev.Kind = expand.NameRef
			}
			prev.Str = s
			return prev
		}
		switch prev.Kind {
		case expand.String:
			prev.Str += s
		case expand.Indexed:
			if len(prev.List) == 0 {
				prev.List = append(prev.List, "")
			}
			prev.List[0] += s
		case expand.Associative:
			// TODO
		}
		return prev
	}
	if as.Array == nil {
		// don't return the zero value, as that's an unset variable
		prev.Kind = expand.String
		if valType == "-n" {
			prev.Kind = expand.NameRef
		}
		prev.Str = ""
		return prev
	}
	elems := as.Array.Elems
	if valType == "" {
		valType = "-a" // indexed
		if len(elems) > 0 && stringIndex(elems[0].Index) {
			valType = "-A" // associative
		}
	}
	if valType == "-A" {
		amap := make(map[string]string, len(elems))
		for _, elem := range elems {
			k := r.literal(elem.Index.(*syntax.Word))
			amap[k] = r.literal(elem.Value)
		}
		if !as.Append {
			prev.Kind = expand.Associative
			prev.Map = amap
			return prev
		}
		// TODO
		return prev
	}
	maxIndex := len(elems) - 1
	indexes := make([]int, len(elems))
	for i, elem := range elems {
		if elem.Index == nil {
			indexes[i] = i
			continue
		}
		k := r.arithm(elem.Index)
		indexes[i] = k
		if k > maxIndex {
			maxIndex = k
		}
	}
	strs := make([]string, maxIndex+1)
	for i, elem := range elems {
		strs[indexes[i]] = r.literal(elem.Value)
	}
	if !as.Append {
		prev.Kind = expand.Indexed
		prev.List = strs
		return prev
	}
	switch prev.Kind {
	case expand.String:
		prev.Kind = expand.Indexed
		prev.List = append([]string{prev.Str}, strs...)
	case expand.Indexed:
		prev.List = append(prev.List, strs...)
	case expand.Associative:
		// TODO
	}
	return prev
}
