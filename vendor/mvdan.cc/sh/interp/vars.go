// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package interp

import (
	"runtime"
	"sort"
	"strings"

	"mvdan.cc/sh/syntax"
)

type Variable struct {
	Local    bool
	Exported bool
	ReadOnly bool
	NameRef  bool
	Value    VarValue
}

// VarValue is one of:
//
//     StringVal
//     IndexArray
//     AssocArray
type VarValue interface{}

type StringVal string

type IndexArray []string

type AssocArray map[string]string

func (r *Runner) lookupVar(name string) (Variable, bool) {
	if name == "" {
		panic("variable name must not be empty")
	}
	if val, e := r.cmdVars[name]; e {
		return Variable{Value: StringVal(val)}, true
	}
	if vr, e := r.funcVars[name]; e {
		return vr, true
	}
	if vr, e := r.Vars[name]; e {
		return vr, true
	}
	if str, e := r.envMap[name]; e {
		return Variable{Value: StringVal(str)}, true
	}
	if runtime.GOOS == "windows" {
		upper := strings.ToUpper(name)
		if str, e := r.envMap[upper]; e {
			return Variable{Value: StringVal(str)}, true
		}
	}
	if r.shellOpts[optNoUnset] {
		r.errf("%s: unbound variable\n", name)
		r.exit = 1
		r.lastExit()
	}
	return Variable{}, false
}

func (r *Runner) getVar(name string) string {
	val, _ := r.lookupVar(name)
	return r.varStr(val, 0)
}

func (r *Runner) delVar(name string) {
	val, _ := r.lookupVar(name)
	if val.ReadOnly {
		r.errf("%s: readonly variable\n", name)
		r.exit = 1
		return
	}
	delete(r.Vars, name)
	delete(r.funcVars, name)
	delete(r.cmdVars, name)
	delete(r.envMap, name)
}

// maxNameRefDepth defines the maximum number of times to follow
// references when expanding a variable. Otherwise, simple name
// reference loops could crash the interpreter quite easily.
const maxNameRefDepth = 100

func (r *Runner) varStr(vr Variable, depth int) string {
	if depth > maxNameRefDepth {
		return ""
	}
	switch x := vr.Value.(type) {
	case StringVal:
		if vr.NameRef {
			vr, _ = r.lookupVar(string(x))
			return r.varStr(vr, depth+1)
		}
		return string(x)
	case IndexArray:
		if len(x) > 0 {
			return x[0]
		}
	case AssocArray:
		// nothing to do
	}
	return ""
}

func (r *Runner) varInd(vr Variable, e syntax.ArithmExpr, depth int) string {
	if depth > maxNameRefDepth {
		return ""
	}
	switch x := vr.Value.(type) {
	case StringVal:
		if vr.NameRef {
			vr, _ = r.lookupVar(string(x))
			return r.varInd(vr, e, depth+1)
		}
		if r.arithm(e) == 0 {
			return string(x)
		}
	case IndexArray:
		switch anyOfLit(e, "@", "*") {
		case "@":
			return strings.Join(x, " ")
		case "*":
			return strings.Join(x, r.ifsJoin)
		}
		i := r.arithm(e)
		if len(x) > 0 {
			return x[i]
		}
	case AssocArray:
		if lit := anyOfLit(e, "@", "*"); lit != "" {
			var strs IndexArray
			keys := make([]string, 0, len(x))
			for k := range x {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				strs = append(strs, x[k])
			}
			if lit == "*" {
				return strings.Join(strs, r.ifsJoin)
			}
			return strings.Join(strs, " ")
		}
		return x[r.loneWord(e.(*syntax.Word))]
	}
	return ""
}

func (r *Runner) setVarString(name, val string) {
	r.setVar(name, nil, Variable{Value: StringVal(val)})
}

func (r *Runner) setVarInternal(name string, vr Variable) {
	if _, ok := vr.Value.(StringVal); ok {
		if r.shellOpts[optAllExport] {
			vr.Exported = true
		}
	} else {
		vr.Exported = false
	}
	if vr.Local {
		if r.funcVars == nil {
			r.funcVars = make(map[string]Variable)
		}
		r.funcVars[name] = vr
	} else {
		r.Vars[name] = vr
	}
	if name == "IFS" {
		r.ifsUpdated()
	}
}

func (r *Runner) setVar(name string, index syntax.ArithmExpr, vr Variable) {
	cur, _ := r.lookupVar(name)
	if cur.ReadOnly {
		r.errf("%s: readonly variable\n", name)
		r.exit = 1
		r.lastExit()
		return
	}
	_, isIndexArray := cur.Value.(IndexArray)
	_, isAssocArray := cur.Value.(AssocArray)

	if _, ok := vr.Value.(StringVal); ok && index == nil {
		// When assigning a string to an array, fall back to the
		// zero value for the index.
		if isIndexArray {
			index = &syntax.Word{Parts: []syntax.WordPart{
				&syntax.Lit{Value: "0"},
			}}
		} else if isAssocArray {
			index = &syntax.Word{Parts: []syntax.WordPart{
				&syntax.DblQuoted{},
			}}
		}
	}
	if index == nil {
		r.setVarInternal(name, vr)
		return
	}

	// from the syntax package, we know that val must be a string if
	// index is non-nil; nested arrays are forbidden.
	valStr := string(vr.Value.(StringVal))

	// if the existing variable is already an AssocArray, try our best
	// to convert the key to a string
	if isAssocArray {
		amap := cur.Value.(AssocArray)
		w, ok := index.(*syntax.Word)
		if !ok {
			return
		}
		k := r.loneWord(w)
		amap[k] = valStr
		cur.Value = amap
		r.setVarInternal(name, cur)
		return
	}
	var list IndexArray
	switch x := cur.Value.(type) {
	case StringVal:
		list = append(list, string(x))
	case IndexArray:
		list = x
	case AssocArray: // done above
	}
	k := r.arithm(index)
	for len(list) < k+1 {
		list = append(list, "")
	}
	list[k] = valStr
	cur.Value = list
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

func (r *Runner) assignVal(as *syntax.Assign, valType string) VarValue {
	prev, prevOk := r.lookupVar(as.Name.Value)
	if as.Naked {
		return prev.Value
	}
	if as.Value != nil {
		s := r.loneWord(as.Value)
		if !as.Append || !prevOk {
			return StringVal(s)
		}
		switch x := prev.Value.(type) {
		case StringVal:
			return x + StringVal(s)
		case IndexArray:
			if len(x) == 0 {
				x = append(x, "")
			}
			x[0] += s
			return x
		case AssocArray:
			// TODO
		}
		return StringVal(s)
	}
	if as.Array == nil {
		return nil
	}
	elems := as.Array.Elems
	if valType == "" {
		if len(elems) == 0 || !stringIndex(elems[0].Index) {
			valType = "-a" // indexed
		} else {
			valType = "-A" // associative
		}
	}
	if valType == "-A" {
		// associative array
		amap := AssocArray(make(map[string]string, len(elems)))
		for _, elem := range elems {
			k := r.loneWord(elem.Index.(*syntax.Word))
			amap[k] = r.loneWord(elem.Value)
		}
		if !as.Append || !prevOk {
			return amap
		}
		// TODO
		return amap
	}
	// indexed array
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
		strs[indexes[i]] = r.loneWord(elem.Value)
	}
	if !as.Append || !prevOk {
		return IndexArray(strs)
	}
	switch x := prev.Value.(type) {
	case StringVal:
		prevList := IndexArray([]string{string(x)})
		return append(prevList, strs...)
	case IndexArray:
		return append(x, strs...)
	case AssocArray:
		// TODO
	}
	return IndexArray(strs)
}

func (r *Runner) ifsUpdated() {
	runes := r.getVar("IFS")
	r.ifsJoin = ""
	if len(runes) > 0 {
		r.ifsJoin = runes[:1]
	}
	r.ifsRune = func(r rune) bool {
		for _, r2 := range runes {
			if r == r2 {
				return true
			}
		}
		return false
	}
}

func (r *Runner) namesByPrefix(prefix string) []string {
	var names []string
	for name := range r.envMap {
		if strings.HasPrefix(name, prefix) {
			names = append(names, name)
		}
	}
	for name := range r.Vars {
		if strings.HasPrefix(name, prefix) {
			names = append(names, name)
		}
	}
	return names
}
