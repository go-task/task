// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package interp

import (
	"path"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mvdan/sh/syntax"
)

func (r *Runner) paramExp(pe *syntax.ParamExp) string {
	name := pe.Param.Value
	var val varValue
	set := false
	switch name {
	case "#":
		val = strconv.Itoa(len(r.args))
	case "*", "@":
		val = strings.Join(r.args, " ")
	case "?":
		val = strconv.Itoa(r.exit)
	default:
		if n, err := strconv.Atoi(name); err == nil {
			if i := n - 1; i < len(r.args) {
				val, set = r.args[i], true
			}
		} else {
			val, set = r.lookupVar(name)
		}
	}
	str := varStr(val)
	if pe.Ind != nil {
		str = r.varInd(val, pe.Ind.Expr)
	}
	switch {
	case pe.Length:
		str = strconv.Itoa(utf8.RuneCountInString(str))
	case pe.Excl:
		val, set = r.lookupVar(str)
		str = varStr(val)
	}
	slicePos := func(expr syntax.ArithmExpr) int {
		p := r.arithm(expr)
		if p < 0 {
			p = len(str) + p
			if p < 0 {
				p = len(str)
			}
		} else if p > len(str) {
			p = len(str)
		}
		return p
	}
	if pe.Slice != nil {
		if pe.Slice.Offset != nil {
			offset := slicePos(pe.Slice.Offset)
			str = str[offset:]
		}
		if pe.Slice.Length != nil {
			length := slicePos(pe.Slice.Length)
			str = str[:length]
		}
	}
	if pe.Repl != nil {
		orig := r.loneWord(pe.Repl.Orig)
		with := r.loneWord(pe.Repl.With)
		n := 1
		if pe.Repl.All {
			n = -1
		}
		str = strings.Replace(str, orig, with, n)
	}
	if pe.Exp != nil {
		arg := r.loneWord(pe.Exp.Word)
		switch pe.Exp.Op {
		case syntax.SubstColPlus:
			if str == "" {
				break
			}
			fallthrough
		case syntax.SubstPlus:
			if set {
				str = arg
			}
		case syntax.SubstMinus:
			if set {
				break
			}
			fallthrough
		case syntax.SubstColMinus:
			if str == "" {
				str = arg
			}
		case syntax.SubstQuest:
			if set {
				break
			}
			fallthrough
		case syntax.SubstColQuest:
			if str == "" {
				r.errf("%s", arg)
				r.exit = 1
				r.lastExit()
			}
		case syntax.SubstAssgn:
			if set {
				break
			}
			fallthrough
		case syntax.SubstColAssgn:
			if str == "" {
				r.setVar(name, arg)
				str = arg
			}
		case syntax.RemSmallPrefix:
			str = removePattern(str, arg, false, false)
		case syntax.RemLargePrefix:
			str = removePattern(str, arg, false, true)
		case syntax.RemSmallSuffix:
			str = removePattern(str, arg, true, false)
		case syntax.RemLargeSuffix:
			str = removePattern(str, arg, true, true)
		case syntax.UpperFirst:
			rs := []rune(str)
			if len(rs) > 0 {
				rs[0] = unicode.ToUpper(rs[0])
			}
			str = string(rs)
		case syntax.UpperAll:
			str = strings.ToUpper(str)
		case syntax.LowerFirst:
			rs := []rune(str)
			if len(rs) > 0 {
				rs[0] = unicode.ToLower(rs[0])
			}
			str = string(rs)
		case syntax.LowerAll:
			str = strings.ToLower(str)
		default: // syntax.OtherParamOps
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
				r.errf("unhandled @%s param expansion", arg)
			default:
				r.errf("unexpected @%s param expansion", arg)
			}
		}
	}
	return str
}

func removePattern(str, pattern string, fromEnd, longest bool) string {
	// TODO: really slow to not re-implement path.Match.
	last := str
	s := str
	i := len(str)
	if fromEnd {
		i = 0
	}
	for {
		if m, _ := path.Match(pattern, s); m {
			last = str[i:]
			if fromEnd {
				last = str[:i]
			}
			if longest {
				return last
			}
		}
		if fromEnd {
			if i++; i >= len(str) {
				break
			}
			s = str[i:]
		} else {
			if i--; i < 1 {
				break
			}
			s = str[:i]
		}
	}
	return last
}
