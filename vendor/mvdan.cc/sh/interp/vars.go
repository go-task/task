// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package interp

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strings"

	"mvdan.cc/sh/syntax"
)

type Environ interface {
	Get(name string) (value string, exists bool)
	Set(name, value string)
	Delete(name string)
	Names() []string
	Copy() Environ
}

type mapEnviron struct {
	names  []string
	values map[string]string
}

func (m *mapEnviron) Get(name string) (string, bool) {
	val, ok := m.values[name]
	return val, ok
}

func (m *mapEnviron) Set(name, value string) {
	_, ok := m.values[name]
	if !ok {
		m.names = append(m.names, name)
		sort.Strings(m.names)
	}
	m.values[name] = value
}

func (m *mapEnviron) Delete(name string) {
	if _, ok := m.values[name]; !ok {
		return
	}
	delete(m.values, name)
	for i, iname := range m.names {
		if iname == name {
			m.names = append(m.names[:i], m.names[i+1:]...)
			return
		}
	}
}

func (m *mapEnviron) Names() []string {
	return m.names
}

func (m *mapEnviron) Copy() Environ {
	m2 := &mapEnviron{
		names:  make([]string, len(m.names)),
		values: make(map[string]string, len(m.values)),
	}
	copy(m2.names, m.names)
	for name, val := range m.values {
		m2.values[name] = val
	}
	return m2
}

func execEnv(env Environ) []string {
	names := env.Names()
	list := make([]string, len(names))
	for i, name := range names {
		val, _ := env.Get(name)
		list[i] = name + "=" + val
	}
	return list
}

func EnvFromList(list []string) (Environ, error) {
	m := mapEnviron{
		names:  make([]string, 0, len(list)),
		values: make(map[string]string, len(list)),
	}
	for _, kv := range list {
		i := strings.IndexByte(kv, '=')
		if i < 0 {
			return nil, fmt.Errorf("env not in the form key=value: %q", kv)
		}
		name, val := kv[:i], kv[i+1:]
		if runtime.GOOS == "windows" {
			name = strings.ToUpper(name)
		}
		m.names = append(m.names, name)
		m.values[name] = val
	}
	sort.Strings(m.names)
	return &m, nil
}

type FuncEnviron func(string) string

func (f FuncEnviron) Get(name string) (string, bool) {
	val := f(name)
	return val, val != ""
}

func (f FuncEnviron) Set(name, value string) {}
func (f FuncEnviron) Delete(name string)     {}
func (f FuncEnviron) Names() []string        { return nil }
func (f FuncEnviron) Copy() Environ          { return f }

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
	if str, e := r.Env.Get(name); e {
		return Variable{Value: StringVal(str)}, true
	}
	if runtime.GOOS == "windows" {
		upper := strings.ToUpper(name)
		if str, e := r.Env.Get(upper); e {
			return Variable{Value: StringVal(str)}, true
		}
	}
	if r.opts[optNoUnset] {
		r.errf("%s: unbound variable\n", name)
		r.setErr(ShellExitStatus(1))
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
	r.Env.Delete(name)
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

func (r *Runner) varInd(ctx context.Context, vr Variable, e syntax.ArithmExpr, depth int) string {
	if depth > maxNameRefDepth {
		return ""
	}
	switch x := vr.Value.(type) {
	case StringVal:
		if vr.NameRef {
			vr, _ = r.lookupVar(string(x))
			return r.varInd(ctx, vr, e, depth+1)
		}
		if r.arithm(ctx, e) == 0 {
			return string(x)
		}
	case IndexArray:
		switch anyOfLit(e, "@", "*") {
		case "@":
			return strings.Join(x, " ")
		case "*":
			return strings.Join(x, r.ifsJoin)
		}
		i := r.arithm(ctx, e)
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
		return x[r.loneWord(ctx, e.(*syntax.Word))]
	}
	return ""
}

func (r *Runner) setVarString(ctx context.Context, name, val string) {
	r.setVar(ctx, name, nil, Variable{Value: StringVal(val)})
}

func (r *Runner) setVarInternal(name string, vr Variable) {
	if _, ok := vr.Value.(StringVal); ok {
		if r.opts[optAllExport] {
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

func (r *Runner) setVar(ctx context.Context, name string, index syntax.ArithmExpr, vr Variable) {
	cur, _ := r.lookupVar(name)
	if cur.ReadOnly {
		r.errf("%s: readonly variable\n", name)
		r.exit = 1
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
		k := r.loneWord(ctx, w)
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
	k := r.arithm(ctx, index)
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

func (r *Runner) assignVal(ctx context.Context, as *syntax.Assign, valType string) VarValue {
	prev, prevOk := r.lookupVar(as.Name.Value)
	if as.Naked {
		return prev.Value
	}
	if as.Value != nil {
		s := r.loneWord(ctx, as.Value)
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
			k := r.loneWord(ctx, elem.Index.(*syntax.Word))
			amap[k] = r.loneWord(ctx, elem.Value)
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
		k := r.arithm(ctx, elem.Index)
		indexes[i] = k
		if k > maxIndex {
			maxIndex = k
		}
	}
	strs := make([]string, maxIndex+1)
	for i, elem := range elems {
		strs[indexes[i]] = r.loneWord(ctx, elem.Value)
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
	for _, name := range r.Env.Names() {
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
