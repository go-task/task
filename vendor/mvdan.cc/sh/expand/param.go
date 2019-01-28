// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package expand

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"mvdan.cc/sh/syntax"
)

func nodeLit(node syntax.Node) string {
	if word, ok := node.(*syntax.Word); ok {
		return word.Lit()
	}
	return ""
}

type UnsetParameterError struct {
	Node    *syntax.ParamExp
	Message string
}

func (u UnsetParameterError) Error() string {
	return u.Message
}

func (cfg *Config) paramExp(pe *syntax.ParamExp) (string, error) {
	oldParam := cfg.curParam
	cfg.curParam = pe
	defer func() { cfg.curParam = oldParam }()

	name := pe.Param.Value
	index := pe.Index
	switch name {
	case "@", "*":
		index = &syntax.Word{Parts: []syntax.WordPart{
			&syntax.Lit{Value: name},
		}}
	}
	var vr Variable
	switch name {
	case "LINENO":
		// This is the only parameter expansion that the environment
		// interface cannot satisfy.
		line := uint64(cfg.curParam.Pos().Line())
		vr.Value = strconv.FormatUint(line, 10)
	default:
		vr = cfg.Env.Get(name)
	}
	orig := vr
	_, vr = vr.Resolve(cfg.Env)
	str, err := cfg.varInd(vr, index)
	if err != nil {
		return "", err
	}
	slicePos := func(n int) int {
		if n < 0 {
			n = len(str) + n
			if n < 0 {
				n = len(str)
			}
		} else if n > len(str) {
			n = len(str)
		}
		return n
	}
	elems := []string{str}
	switch nodeLit(index) {
	case "@", "*":
		switch x := vr.Value.(type) {
		case nil:
			elems = nil
		case []string:
			elems = x
		}
	}
	switch {
	case pe.Length:
		n := len(elems)
		switch nodeLit(index) {
		case "@", "*":
		default:
			n = utf8.RuneCountInString(str)
		}
		str = strconv.Itoa(n)
	case pe.Excl:
		var strs []string
		if pe.Names != 0 {
			strs = cfg.namesByPrefix(pe.Param.Value)
		} else if orig.NameRef {
			strs = append(strs, orig.Value.(string))
		} else if x, ok := vr.Value.([]string); ok {
			for i, e := range x {
				if e != "" {
					strs = append(strs, strconv.Itoa(i))
				}
			}
		} else if x, ok := vr.Value.(map[string]string); ok {
			for k := range x {
				strs = append(strs, k)
			}
		} else if str != "" {
			vr = cfg.Env.Get(str)
			strs = append(strs, vr.String())
		}
		sort.Strings(strs)
		str = strings.Join(strs, " ")
	case pe.Slice != nil:
		if pe.Slice.Offset != nil {
			n, err := Arithm(cfg, pe.Slice.Offset)
			if err != nil {
				return "", err
			}
			str = str[slicePos(n):]
		}
		if pe.Slice.Length != nil {
			n, err := Arithm(cfg, pe.Slice.Length)
			if err != nil {
				return "", err
			}
			str = str[:slicePos(n)]
		}
	case pe.Repl != nil:
		orig, err := Pattern(cfg, pe.Repl.Orig)
		if err != nil {
			return "", err
		}
		with, err := Literal(cfg, pe.Repl.With)
		if err != nil {
			return "", err
		}
		n := 1
		if pe.Repl.All {
			n = -1
		}
		locs := findAllIndex(orig, str, n)
		buf := cfg.strBuilder()
		last := 0
		for _, loc := range locs {
			buf.WriteString(str[last:loc[0]])
			buf.WriteString(with)
			last = loc[1]
		}
		buf.WriteString(str[last:])
		str = buf.String()
	case pe.Exp != nil:
		arg, err := Literal(cfg, pe.Exp.Word)
		if err != nil {
			return "", err
		}
		switch op := pe.Exp.Op; op {
		case syntax.SubstColPlus:
			if str == "" {
				break
			}
			fallthrough
		case syntax.SubstPlus:
			if vr.IsSet() {
				str = arg
			}
		case syntax.SubstMinus:
			if vr.IsSet() {
				break
			}
			fallthrough
		case syntax.SubstColMinus:
			if str == "" {
				str = arg
			}
		case syntax.SubstQuest:
			if vr.IsSet() {
				break
			}
			fallthrough
		case syntax.SubstColQuest:
			if str == "" {
				return "", UnsetParameterError{
					Node:    pe,
					Message: arg,
				}
			}
		case syntax.SubstAssgn:
			if vr.IsSet() {
				break
			}
			fallthrough
		case syntax.SubstColAssgn:
			if str == "" {
				cfg.envSet(name, arg)
				str = arg
			}
		case syntax.RemSmallPrefix, syntax.RemLargePrefix,
			syntax.RemSmallSuffix, syntax.RemLargeSuffix:
			suffix := op == syntax.RemSmallSuffix ||
				op == syntax.RemLargeSuffix
			large := op == syntax.RemLargePrefix ||
				op == syntax.RemLargeSuffix
			for i, elem := range elems {
				elems[i] = removePattern(elem, arg, suffix, large)
			}
			str = strings.Join(elems, " ")
		case syntax.UpperFirst, syntax.UpperAll,
			syntax.LowerFirst, syntax.LowerAll:

			caseFunc := unicode.ToLower
			if op == syntax.UpperFirst || op == syntax.UpperAll {
				caseFunc = unicode.ToUpper
			}
			all := op == syntax.UpperAll || op == syntax.LowerAll

			// empty string means '?'; nothing to do there
			expr, err := syntax.TranslatePattern(arg, false)
			if err != nil {
				return str, nil
			}
			rx := regexp.MustCompile(expr)

			for i, elem := range elems {
				rs := []rune(elem)
				for ri, r := range rs {
					if rx.MatchString(string(r)) {
						rs[ri] = caseFunc(r)
						if !all {
							break
						}
					}
				}
				elems[i] = string(rs)
			}
			str = strings.Join(elems, " ")
		case syntax.OtherParamOps:
			switch arg {
			case "Q":
				str = strconv.Quote(str)
			case "E":
				tail := str
				var rns []rune
				for tail != "" {
					var rn rune
					rn, _, tail, _ = strconv.UnquoteChar(tail, 0)
					rns = append(rns, rn)
				}
				str = string(rns)
			case "P", "A", "a":
				panic(fmt.Sprintf("unhandled @%s param expansion", arg))
			default:
				panic(fmt.Sprintf("unexpected @%s param expansion", arg))
			}
		}
	}
	return str, nil
}

func removePattern(str, pattern string, fromEnd, greedy bool) string {
	expr, err := syntax.TranslatePattern(pattern, greedy)
	if err != nil {
		return str
	}
	switch {
	case fromEnd && !greedy:
		// use .* to get the right-most (shortest) match
		expr = ".*(" + expr + ")$"
	case fromEnd:
		// simple suffix
		expr = "(" + expr + ")$"
	default:
		// simple prefix
		expr = "^(" + expr + ")"
	}
	// no need to check error as TranslatePattern returns one
	rx := regexp.MustCompile(expr)
	if loc := rx.FindStringSubmatchIndex(str); loc != nil {
		// remove the original pattern (the submatch)
		str = str[:loc[2]] + str[loc[3]:]
	}
	return str
}

func (cfg *Config) varInd(vr Variable, idx syntax.ArithmExpr) (string, error) {
	if idx == nil {
		return vr.String(), nil
	}
	switch x := vr.Value.(type) {
	case string:
		n, err := Arithm(cfg, idx)
		if err != nil {
			return "", err
		}
		if n == 0 {
			return x, nil
		}
	case []string:
		switch nodeLit(idx) {
		case "@":
			return strings.Join(x, " "), nil
		case "*":
			return cfg.ifsJoin(x), nil
		}
		i, err := Arithm(cfg, idx)
		if err != nil {
			return "", err
		}
		if len(x) > 0 {
			return x[i], nil
		}
	case map[string]string:
		switch lit := nodeLit(idx); lit {
		case "@", "*":
			var strs []string
			keys := make([]string, 0, len(x))
			for k := range x {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				strs = append(strs, x[k])
			}
			if lit == "*" {
				return cfg.ifsJoin(strs), nil
			}
			return strings.Join(strs, " "), nil
		}
		val, err := Literal(cfg, idx.(*syntax.Word))
		if err != nil {
			return "", err
		}
		return x[val], nil
	}
	return "", nil
}

func (cfg *Config) namesByPrefix(prefix string) []string {
	var names []string
	cfg.Env.Each(func(name string, vr Variable) bool {
		if strings.HasPrefix(name, prefix) {
			names = append(names, name)
		}
		return true
	})
	return names
}
