// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package interp

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"

	"golang.org/x/crypto/ssh/terminal"

	"mvdan.cc/sh/syntax"
)

// non-empty string is true, empty string is false
func (r *Runner) bashTest(expr syntax.TestExpr) string {
	switch x := expr.(type) {
	case *syntax.Word:
		return r.loneWord(x)
	case *syntax.ParenTest:
		return r.bashTest(x.X)
	case *syntax.BinaryTest:
		switch x.Op {
		case syntax.TsMatch, syntax.TsNoMatch:
			str := r.loneWord(x.X.(*syntax.Word))
			yw := x.Y.(*syntax.Word)
			pat := r.lonePattern(yw)
			if match(pat, str) == (x.Op == syntax.TsMatch) {
				return "1"
			}
			return ""
		}
		if r.binTest(x.Op, r.bashTest(x.X), r.bashTest(x.Y)) {
			return "1"
		}
		return ""
	case *syntax.UnaryTest:
		if r.unTest(x.Op, r.bashTest(x.X)) {
			return "1"
		}
		return ""
	}
	return ""
}

func (r *Runner) binTest(op syntax.BinTestOperator, x, y string) bool {
	switch op {
	case syntax.TsReMatch:
		re, err := regexp.Compile(y)
		if err != nil {
			r.exit = 2
			return false
		}
		return re.MatchString(x)
	case syntax.TsNewer:
		info1, err1 := r.stat(x)
		info2, err2 := r.stat(y)
		if err1 != nil || err2 != nil {
			return false
		}
		return info1.ModTime().After(info2.ModTime())
	case syntax.TsOlder:
		info1, err1 := r.stat(x)
		info2, err2 := r.stat(y)
		if err1 != nil || err2 != nil {
			return false
		}
		return info1.ModTime().Before(info2.ModTime())
	case syntax.TsDevIno:
		info1, err1 := r.stat(x)
		info2, err2 := r.stat(y)
		if err1 != nil || err2 != nil {
			return false
		}
		return os.SameFile(info1, info2)
	case syntax.TsEql:
		return atoi(x) == atoi(y)
	case syntax.TsNeq:
		return atoi(x) != atoi(y)
	case syntax.TsLeq:
		return atoi(x) <= atoi(y)
	case syntax.TsGeq:
		return atoi(x) >= atoi(y)
	case syntax.TsLss:
		return atoi(x) < atoi(y)
	case syntax.TsGtr:
		return atoi(x) > atoi(y)
	case syntax.AndTest:
		return x != "" && y != ""
	case syntax.OrTest:
		return x != "" || y != ""
	case syntax.TsBefore:
		return x < y
	default: // syntax.TsAfter
		return x > y
	}
}

func (r *Runner) statMode(name string, mode os.FileMode) bool {
	info, err := r.stat(name)
	return err == nil && info.Mode()&mode != 0
}

func (r *Runner) unTest(op syntax.UnTestOperator, x string) bool {
	switch op {
	case syntax.TsExists:
		_, err := r.stat(x)
		return err == nil
	case syntax.TsRegFile:
		info, err := r.stat(x)
		return err == nil && info.Mode().IsRegular()
	case syntax.TsDirect:
		return r.statMode(x, os.ModeDir)
	case syntax.TsCharSp:
		return r.statMode(x, os.ModeCharDevice)
	case syntax.TsBlckSp:
		info, err := r.stat(x)
		return err == nil && info.Mode()&os.ModeDevice != 0 &&
			info.Mode()&os.ModeCharDevice == 0
	case syntax.TsNmPipe:
		return r.statMode(x, os.ModeNamedPipe)
	case syntax.TsSocket:
		return r.statMode(x, os.ModeSocket)
	case syntax.TsSmbLink:
		info, err := os.Lstat(r.relPath(x))
		return err == nil && info.Mode()&os.ModeSymlink != 0
	case syntax.TsSticky:
		return r.statMode(x, os.ModeSticky)
	case syntax.TsUIDSet:
		return r.statMode(x, os.ModeSetuid)
	case syntax.TsGIDSet:
		return r.statMode(x, os.ModeSetgid)
	//case syntax.TsGrpOwn:
	//case syntax.TsUsrOwn:
	//case syntax.TsModif:
	case syntax.TsRead:
		f, err := r.open(r.relPath(x), os.O_RDONLY, 0, false)
		if err == nil {
			f.Close()
		}
		return err == nil
	case syntax.TsWrite:
		f, err := r.open(r.relPath(x), os.O_WRONLY, 0, false)
		if err == nil {
			f.Close()
		}
		return err == nil
	case syntax.TsExec:
		_, err := exec.LookPath(r.relPath(x))
		return err == nil
	case syntax.TsNoEmpty:
		info, err := r.stat(x)
		return err == nil && info.Size() > 0
	case syntax.TsFdTerm:
		return terminal.IsTerminal(atoi(x))
	case syntax.TsEmpStr:
		return x == ""
	case syntax.TsNempStr:
		return x != ""
	case syntax.TsOptSet:
		if opt := r.optByName(x, false); opt != nil {
			return *opt
		}
		return false
	case syntax.TsVarSet:
		_, e := r.lookupVar(x)
		return e
	case syntax.TsRefVar:
		v, _ := r.lookupVar(x)
		return v.NameRef
	case syntax.TsNot:
		return x == ""
	default:
		panic(fmt.Sprintf("unhandled unary test op: %v", op))
	}
}
