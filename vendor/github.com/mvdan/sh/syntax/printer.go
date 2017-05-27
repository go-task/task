// Copyright (c) 2016, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package syntax

import (
	"bufio"
	"io"
	"strings"
)

func Indent(spaces int) func(*Printer) {
	return func(p *Printer) { p.indentSpaces = spaces }
}

func BinaryNextLine(p *Printer) { p.binNextLine = true }

func NewPrinter(options ...func(*Printer)) *Printer {
	p := &Printer{
		bufWriter:  bufio.NewWriter(nil),
		lenPrinter: new(Printer),
	}
	for _, opt := range options {
		opt(p)
	}
	return p
}

// Print "pretty-prints" the given AST file to the given writer.
func (p *Printer) Print(w io.Writer, f *File) error {
	p.reset()
	p.lines, p.comments = f.lines, f.Comments
	p.bufWriter.Reset(w)
	p.stmts(f.Stmts)
	p.commentsUpTo(0)
	p.newline(0)
	return p.bufWriter.Flush()
}

type bufWriter interface {
	WriteByte(byte) error
	WriteString(string) (int, error)
	Reset(io.Writer)
	Flush() error
}

type Printer struct {
	bufWriter

	indentSpaces int
	binNextLine  bool

	lines []Pos

	wantSpace   bool
	wantNewline bool
	wroteSemi   bool

	commentPadding int

	// nline is the position of the next newline
	nline      Pos
	nlineIndex int

	// lastLevel is the last level of indentation that was used.
	lastLevel int
	// level is the current level of indentation.
	level int
	// levelIncs records which indentation level increments actually
	// took place, to revert them once their section ends.
	levelIncs []bool

	nestedBinary bool

	// comments is the list of pending comments to write.
	comments []*Comment

	// pendingHdocs is the list of pending heredocs to write.
	pendingHdocs []*Redirect

	// used in stmtCols to align comments
	lenPrinter *Printer
	lenCounter byteCounter
}

func (p *Printer) reset() {
	p.wantSpace, p.wantNewline = false, false
	p.commentPadding = 0
	p.nline, p.nlineIndex = 0, 0
	p.lastLevel, p.level = 0, 0
	p.levelIncs = p.levelIncs[:0]
	p.nestedBinary = false
	p.pendingHdocs = p.pendingHdocs[:0]
}

func (p *Printer) incLine() {
	if p.nlineIndex++; p.nlineIndex >= len(p.lines) {
		p.nline = maxPos
	} else {
		p.nline = p.lines[p.nlineIndex]
	}
}

func (p *Printer) incLines(pos Pos) {
	for p.nline < pos {
		p.incLine()
	}
}

func (p *Printer) spaces(n int) {
	for i := 0; i < n; i++ {
		p.WriteByte(' ')
	}
}

func (p *Printer) bslashNewl() {
	if p.wantSpace {
		p.WriteByte(' ')
	}
	p.WriteString("\\\n")
	p.wantSpace = false
	p.incLine()
}

func (p *Printer) spacedString(s string) {
	if p.wantSpace {
		p.WriteByte(' ')
	}
	p.WriteString(s)
	p.wantSpace = true
}

func (p *Printer) semiOrNewl(s string, pos Pos) {
	if p.wantNewline {
		p.newline(pos)
		p.indent()
	} else {
		if !p.wroteSemi {
			p.WriteByte(';')
		}
		p.WriteByte(' ')
		p.incLines(pos)
	}
	p.WriteString(s)
	p.wantSpace = true
}

func (p *Printer) incLevel() {
	inc := false
	if p.level <= p.lastLevel || len(p.levelIncs) == 0 {
		p.level++
		inc = true
	} else if last := &p.levelIncs[len(p.levelIncs)-1]; *last {
		*last = false
		inc = true
	}
	p.levelIncs = append(p.levelIncs, inc)
}

func (p *Printer) decLevel() {
	if p.levelIncs[len(p.levelIncs)-1] {
		p.level--
	}
	p.levelIncs = p.levelIncs[:len(p.levelIncs)-1]
}

func (p *Printer) indent() {
	p.lastLevel = p.level
	switch {
	case p.level == 0:
	case p.indentSpaces == 0:
		for i := 0; i < p.level; i++ {
			p.WriteByte('\t')
		}
	case p.indentSpaces > 0:
		p.spaces(p.indentSpaces * p.level)
	}
}

func (p *Printer) newline(pos Pos) {
	p.wantNewline, p.wantSpace = false, false
	p.WriteByte('\n')
	if pos > p.nline {
		p.incLine()
	}
	hdocs := p.pendingHdocs
	p.pendingHdocs = p.pendingHdocs[:0]
	for _, r := range hdocs {
		if r.Hdoc != nil {
			p.word(r.Hdoc)
			p.incLines(r.Hdoc.End())
		}
		p.unquotedWord(r.Word)
		p.WriteByte('\n')
		p.incLine()
		p.wantSpace = false
	}
}

func (p *Printer) newlines(pos Pos) {
	p.newline(pos)
	if pos > p.nline {
		// preserve single empty lines
		p.WriteByte('\n')
		p.incLine()
	}
	p.indent()
}

func (p *Printer) commentsAndSeparate(pos Pos) {
	p.commentsUpTo(pos)
	if p.wantNewline || pos > p.nline {
		p.newlines(pos)
	}
}

func (p *Printer) sepTok(s string, pos Pos) {
	p.level++
	p.commentsUpTo(pos)
	p.level--
	if p.wantNewline || pos > p.nline {
		p.newlines(pos)
	}
	p.WriteString(s)
	p.wantSpace = true
}

func (p *Printer) semiRsrv(s string, pos Pos, fallback bool) {
	p.level++
	p.commentsUpTo(pos)
	p.level--
	if p.wantNewline || pos > p.nline {
		p.newlines(pos)
	} else {
		if fallback && !p.wroteSemi {
			p.WriteByte(';')
		}
		if p.wantSpace {
			p.WriteByte(' ')
		}
	}
	p.WriteString(s)
	p.wantSpace = true
}

func (p *Printer) anyCommentsBefore(pos Pos) bool {
	if !pos.IsValid() || len(p.comments) < 1 {
		return false
	}
	return p.comments[0].Hash < pos
}

func (p *Printer) commentsUpTo(pos Pos) {
	if len(p.comments) < 1 {
		return
	}
	c := p.comments[0]
	if pos.IsValid() && c.Hash >= pos {
		return
	}
	p.comments = p.comments[1:]
	switch {
	case p.nlineIndex == 0:
	case c.Hash > p.nline:
		p.newlines(c.Hash)
	case p.wantSpace:
		p.spaces(p.commentPadding + 1)
	}
	p.incLines(c.Hash)
	p.WriteByte('#')
	p.WriteString(c.Text)
	p.commentsUpTo(pos)
}

func (p *Printer) wordPart(wp WordPart) {
	switch x := wp.(type) {
	case *Lit:
		p.WriteString(x.Value)
	case *SglQuoted:
		if x.Dollar {
			p.WriteByte('$')
		}
		p.WriteByte('\'')
		p.WriteString(x.Value)
		p.WriteByte('\'')
		p.incLines(x.End())
	case *DblQuoted:
		p.dblQuoted(x)
	case *CmdSubst:
		p.incLines(x.Pos())
		switch {
		case x.MirBSDTempFile:
			p.WriteString("${")
			p.wantSpace = true
			p.nestedStmts(x.Stmts, x.Right)
			p.wantSpace = false
			p.semiRsrv("}", x.Right, true)
		case x.MirBSDReplyVar:
			p.WriteString("${|")
			p.nestedStmts(x.Stmts, x.Right)
			p.wantSpace = false
			p.semiRsrv("}", x.Right, true)
		default:
			p.WriteString("$(")
			p.wantSpace = len(x.Stmts) > 0 && startsWithLparen(x.Stmts[0])
			p.nestedStmts(x.Stmts, x.Right)
			p.sepTok(")", x.Right)
		}
	case *ParamExp:
		p.paramExp(x)
	case *ArithmExp:
		p.WriteString("$((")
		if x.Unsigned {
			p.WriteString("# ")
		}
		p.arithmExpr(x.X, false, false)
		p.WriteString("))")
	case *ExtGlob:
		p.WriteString(x.Op.String())
		p.WriteString(x.Pattern.Value)
		p.WriteByte(')')
	case *ProcSubst:
		// avoid conflict with << and others
		if p.wantSpace {
			p.WriteByte(' ')
			p.wantSpace = false
		}
		p.WriteString(x.Op.String())
		p.nestedStmts(x.Stmts, 0)
		p.WriteByte(')')
	}
}

func (p *Printer) dblQuoted(dq *DblQuoted) {
	if dq.Dollar {
		p.WriteByte('$')
	}
	p.WriteByte('"')
	for i, n := range dq.Parts {
		p.wordPart(n)
		if i == len(dq.Parts)-1 {
			p.incLines(n.End())
		}
	}
	p.WriteByte('"')
}

func (p *Printer) wroteIndex(index ArithmExpr, key *DblQuoted) bool {
	if index == nil && key == nil {
		return false
	}
	p.WriteByte('[')
	if index != nil {
		p.arithmExpr(index, false, false)
	} else {
		p.dblQuoted(key)
	}
	p.WriteByte(']')
	return true
}

func (p *Printer) paramExp(pe *ParamExp) {
	if pe.nakedIndex() { // arr[x]
		p.WriteString(pe.Param.Value)
		p.wroteIndex(pe.Index, pe.Key)
		return
	}
	if pe.Short { // $var
		p.WriteByte('$')
		p.WriteString(pe.Param.Value)
		return
	}
	// ${var...}
	p.WriteString("${")
	switch {
	case pe.Length:
		p.WriteByte('#')
	case pe.Width:
		p.WriteByte('%')
	case pe.Indirect:
		p.WriteByte('!')
	}
	if pe.Param != nil {
		p.WriteString(pe.Param.Value)
	}
	p.wroteIndex(pe.Index, pe.Key)
	if pe.Slice != nil {
		p.WriteByte(':')
		p.arithmExpr(pe.Slice.Offset, true, true)
		if pe.Slice.Length != nil {
			p.WriteByte(':')
			p.arithmExpr(pe.Slice.Length, true, false)
		}
	} else if pe.Repl != nil {
		if pe.Repl.All {
			p.WriteByte('/')
		}
		p.WriteByte('/')
		if pe.Repl.Orig != nil {
			p.word(pe.Repl.Orig)
		}
		p.WriteByte('/')
		if pe.Repl.With != nil {
			p.word(pe.Repl.With)
		}
	} else if pe.Exp != nil {
		p.WriteString(pe.Exp.Op.String())
		if pe.Exp.Word != nil {
			p.word(pe.Exp.Word)
		}
	}
	p.WriteByte('}')
}

func (p *Printer) loop(loop Loop) {
	switch x := loop.(type) {
	case *WordIter:
		p.WriteString(x.Name.Value)
		if len(x.Items) > 0 {
			p.spacedString(" in")
			p.wordJoin(x.Items)
		}
	case *CStyleLoop:
		p.WriteString("((")
		if x.Init == nil {
			p.WriteByte(' ')
		}
		p.arithmExpr(x.Init, false, false)
		p.WriteString("; ")
		p.arithmExpr(x.Cond, false, false)
		p.WriteString("; ")
		p.arithmExpr(x.Post, false, false)
		p.WriteString("))")
	}
}

func (p *Printer) arithmExpr(expr ArithmExpr, compact, spacePlusMinus bool) {
	switch x := expr.(type) {
	case *Word:
		p.word(x)
	case *BinaryArithm:
		if compact {
			p.arithmExpr(x.X, compact, spacePlusMinus)
			p.WriteString(x.Op.String())
			p.arithmExpr(x.Y, compact, false)
		} else {
			p.arithmExpr(x.X, compact, spacePlusMinus)
			if x.Op != Comma {
				p.WriteByte(' ')
			}
			p.WriteString(x.Op.String())
			p.WriteByte(' ')
			p.arithmExpr(x.Y, compact, false)
		}
	case *UnaryArithm:
		if x.Post {
			p.arithmExpr(x.X, compact, spacePlusMinus)
			p.WriteString(x.Op.String())
		} else {
			if spacePlusMinus {
				switch x.Op {
				case Plus, Minus:
					p.WriteByte(' ')
				}
			}
			p.WriteString(x.Op.String())
			p.arithmExpr(x.X, compact, false)
		}
	case *ParenArithm:
		p.WriteByte('(')
		p.arithmExpr(x.X, false, false)
		p.WriteByte(')')
	}
}

func (p *Printer) testExpr(expr TestExpr) {
	switch x := expr.(type) {
	case *Word:
		p.word(x)
	case *BinaryTest:
		p.testExpr(x.X)
		p.WriteByte(' ')
		p.WriteString(x.Op.String())
		p.WriteByte(' ')
		p.testExpr(x.Y)
	case *UnaryTest:
		p.WriteString(x.Op.String())
		p.WriteByte(' ')
		p.testExpr(x.X)
	case *ParenTest:
		p.WriteByte('(')
		p.testExpr(x.X)
		p.WriteByte(')')
	}
}

func (p *Printer) word(w *Word) {
	for _, n := range w.Parts {
		p.wordPart(n)
	}
	p.wantSpace = true
}

func (p *Printer) unquotedWord(w *Word) {
	for _, wp := range w.Parts {
		switch x := wp.(type) {
		case *SglQuoted:
			p.WriteString(x.Value)
		case *DblQuoted:
			for _, qp := range x.Parts {
				p.wordPart(qp)
			}
		case *Lit:
			for i := 0; i < len(x.Value); i++ {
				if b := x.Value[i]; b == '\\' {
					if i++; i < len(x.Value) {
						p.WriteByte(x.Value[i])
					}
				} else {
					p.WriteByte(b)
				}
			}
		}
	}
}

func (p *Printer) wordJoin(ws []*Word) {
	anyNewline := false
	for _, w := range ws {
		if pos := w.Pos(); pos > p.nline {
			p.commentsUpTo(pos)
			p.bslashNewl()
			if !anyNewline {
				p.incLevel()
				anyNewline = true
			}
			p.indent()
		} else if p.wantSpace {
			p.WriteByte(' ')
			p.wantSpace = false
		}
		p.word(w)
	}
	if anyNewline {
		p.decLevel()
	}
}

func (p *Printer) elemJoin(elems []*ArrayElem) {
	anyNewline := false
	for _, el := range elems {
		if pos := el.Pos(); pos > p.nline {
			p.commentsUpTo(pos)
			p.WriteByte('\n')
			p.incLine()
			if !anyNewline {
				p.incLevel()
				anyNewline = true
			}
			p.indent()
		} else if p.wantSpace {
			p.WriteByte(' ')
			p.wantSpace = false
		}
		if p.wroteIndex(el.Index, el.Key) {
			p.WriteByte('=')
		}
		p.word(el.Value)
	}
	if anyNewline {
		p.decLevel()
	}
}

func (p *Printer) stmt(s *Stmt) {
	if s.Negated {
		p.spacedString("!")
	}
	p.assigns(s.Assigns, true)
	var startRedirs int
	if s.Cmd != nil {
		startRedirs = p.command(s.Cmd, s.Redirs)
	}
	anyNewline := false
	for _, r := range s.Redirs[startRedirs:] {
		if r.OpPos > p.nline {
			p.bslashNewl()
			if !anyNewline {
				p.incLevel()
				anyNewline = true
			}
			p.indent()
		}
		p.commentsAndSeparate(r.OpPos)
		if p.wantSpace {
			p.WriteByte(' ')
		}
		if r.N != nil {
			p.WriteString(r.N.Value)
		}
		p.WriteString(r.Op.String())
		p.wantSpace = true
		p.word(r.Word)
		if r.Op == Hdoc || r.Op == DashHdoc {
			p.pendingHdocs = append(p.pendingHdocs, r)
		}
	}
	p.wroteSemi = false
	switch {
	case s.Semicolon.IsValid() && s.Semicolon > p.nline:
		p.incLevel()
		p.bslashNewl()
		p.indent()
		p.decLevel()
		p.WriteByte(';')
		p.wroteSemi = true
	case s.Background:
		p.WriteString(" &")
	case s.Coprocess:
		p.WriteString(" |&")
	}
	if anyNewline {
		p.decLevel()
	}
}

func (p *Printer) command(cmd Command, redirs []*Redirect) (startRedirs int) {
	if p.wantSpace {
		p.WriteByte(' ')
		p.wantSpace = false
	}
	switch x := cmd.(type) {
	case *CallExpr:
		if len(x.Args) <= 1 {
			p.wordJoin(x.Args)
			return 0
		}
		p.wordJoin(x.Args[:1])
		for _, r := range redirs {
			if r.Pos() > x.Args[1].Pos() || r.Op == Hdoc || r.Op == DashHdoc {
				break
			}
			if p.wantSpace {
				p.WriteByte(' ')
			}
			if r.N != nil {
				p.WriteString(r.N.Value)
			}
			p.WriteString(r.Op.String())
			p.wantSpace = true
			p.word(r.Word)
			startRedirs++
		}
		p.wordJoin(x.Args[1:])
	case *Block:
		p.WriteByte('{')
		p.wantSpace = true
		p.nestedStmts(x.Stmts, x.Rbrace)
		p.semiRsrv("}", x.Rbrace, true)
	case *IfClause:
		p.spacedString("if")
		p.nestedStmts(x.CondStmts, 0)
		p.semiOrNewl("then", x.Then)
		p.nestedStmts(x.ThenStmts, 0)
		for _, el := range x.Elifs {
			p.semiRsrv("elif", el.Elif, true)
			p.nestedStmts(el.CondStmts, 0)
			p.semiOrNewl("then", el.Then)
			p.nestedStmts(el.ThenStmts, 0)
		}
		if len(x.ElseStmts) > 0 {
			p.semiRsrv("else", x.Else, true)
			p.nestedStmts(x.ElseStmts, 0)
		} else if x.Else.IsValid() {
			p.incLines(x.Else)
		}
		p.semiRsrv("fi", x.Fi, true)
	case *Subshell:
		p.WriteByte('(')
		p.wantSpace = len(x.Stmts) > 0 && startsWithLparen(x.Stmts[0])
		p.nestedStmts(x.Stmts, x.Rparen)
		p.sepTok(")", x.Rparen)
	case *WhileClause:
		if x.Until {
			p.spacedString("until")
		} else {
			p.spacedString("while")
		}
		p.nestedStmts(x.CondStmts, 0)
		p.semiOrNewl("do", x.Do)
		p.nestedStmts(x.DoStmts, 0)
		p.semiRsrv("done", x.Done, true)
	case *ForClause:
		p.WriteString("for ")
		p.loop(x.Loop)
		p.semiOrNewl("do", x.Do)
		p.nestedStmts(x.DoStmts, 0)
		p.semiRsrv("done", x.Done, true)
	case *BinaryCmd:
		p.stmt(x.X)
		if x.Y.Pos() < p.nline {
			// leave p.nestedBinary untouched
			p.spacedString(x.Op.String())
			p.stmt(x.Y)
			break
		}
		indent := !p.nestedBinary
		if indent {
			p.incLevel()
		}
		if p.binNextLine {
			if len(p.pendingHdocs) == 0 {
				p.bslashNewl()
				p.indent()
			}
			p.spacedString(x.Op.String())
			if p.anyCommentsBefore(x.Y.Pos()) {
				p.wantSpace = false
				p.WriteByte('\n')
				p.indent()
				p.incLines(p.comments[0].Pos())
				p.commentsUpTo(x.Y.Pos())
				p.WriteByte('\n')
				p.indent()
			}
		} else {
			p.wantSpace = true
			p.spacedString(x.Op.String())
			if x.OpPos > p.nline {
				p.incLines(x.OpPos)
			}
			p.commentsUpTo(x.Y.Pos())
			p.newline(0)
			p.indent()
		}
		p.incLines(x.Y.Pos())
		_, p.nestedBinary = x.Y.Cmd.(*BinaryCmd)
		p.stmt(x.Y)
		if indent {
			p.decLevel()
		}
		p.nestedBinary = false
	case *FuncDecl:
		if x.BashStyle {
			p.WriteString("function ")
		}
		p.WriteString(x.Name.Value)
		p.WriteString("() ")
		p.incLines(x.Body.Pos())
		p.stmt(x.Body)
	case *CaseClause:
		p.WriteString("case ")
		p.word(x.Word)
		p.WriteString(" in")
		for _, ci := range x.Items {
			p.commentsAndSeparate(ci.Patterns[0].Pos())
			for i, w := range ci.Patterns {
				if i > 0 {
					p.spacedString("|")
				}
				if p.wantSpace {
					p.WriteByte(' ')
				}
				p.word(w)
			}
			p.WriteByte(')')
			p.wantSpace = true
			sep := len(ci.Stmts) > 1 || (len(ci.Stmts) > 0 && ci.Stmts[0].Pos() > p.nline)
			p.nestedStmts(ci.Stmts, 0)
			p.level++
			if sep {
				p.commentsUpTo(ci.OpPos)
				p.newlines(ci.OpPos)
			}
			p.spacedString(ci.Op.String())
			p.incLines(ci.OpPos)
			p.level--
			if sep || ci.OpPos == x.Esac {
				p.wantNewline = true
			}
		}
		p.semiRsrv("esac", x.Esac, len(x.Items) == 0)
	case *ArithmCmd:
		p.WriteString("((")
		if x.Unsigned {
			p.WriteString("# ")
		}
		p.arithmExpr(x.X, false, false)
		p.WriteString("))")
	case *TestClause:
		p.WriteString("[[ ")
		p.testExpr(x.X)
		p.spacedString("]]")
	case *DeclClause:
		p.spacedString(x.Variant)
		for _, w := range x.Opts {
			p.WriteByte(' ')
			p.word(w)
		}
		p.assigns(x.Assigns, false)
	case *TimeClause:
		p.spacedString("time")
		if x.Stmt != nil {
			p.stmt(x.Stmt)
		}
	case *CoprocClause:
		p.spacedString("coproc")
		if x.Name != nil {
			p.WriteByte(' ')
			p.WriteString(x.Name.Value)
		}
		p.stmt(x.Stmt)
	case *LetClause:
		p.spacedString("let")
		for _, n := range x.Exprs {
			p.WriteByte(' ')
			p.arithmExpr(n, true, false)
		}
	}
	return startRedirs
}

func startsWithLparen(s *Stmt) bool {
	switch x := s.Cmd.(type) {
	case *Subshell:
		return true
	case *BinaryCmd:
		return startsWithLparen(x.X)
	}
	return false
}

func (p *Printer) hasInline(pos, npos, nline Pos) bool {
	for _, c := range p.comments {
		if c.Hash > nline {
			return false
		}
		if c.Hash > pos && (npos == 0 || c.Hash < npos) {
			return true
		}
	}
	return false
}

func (p *Printer) stmts(stmts []*Stmt) {
	switch len(stmts) {
	case 0:
		return
	case 1:
		s := stmts[0]
		pos := s.Pos()
		p.commentsUpTo(pos)
		if pos <= p.nline {
			p.stmt(s)
		} else {
			if p.nlineIndex > 0 {
				p.newlines(pos)
			}
			p.incLines(pos)
			p.stmt(s)
			p.wantNewline = true
		}
		return
	}
	inlineIndent := 0
	for i, s := range stmts {
		pos := s.Pos()
		ind := p.nlineIndex
		p.commentsUpTo(pos)
		if p.nlineIndex > 0 {
			p.newlines(pos)
		}
		p.incLines(pos)
		p.stmt(s)
		var npos Pos
		if i+1 < len(stmts) {
			npos = stmts[i+1].Pos()
		}
		if !p.hasInline(pos, npos, p.nline) {
			inlineIndent = 0
			p.commentPadding = 0
			continue
		}
		if ind < len(p.lines)-1 && s.End() > p.lines[ind+1] {
			inlineIndent = 0
		}
		if inlineIndent == 0 {
			ind2 := p.nlineIndex
			nline2 := p.nline
			follow := stmts[i:]
			for j, s2 := range follow {
				pos2 := s2.Pos()
				var npos2 Pos
				if j+1 < len(follow) {
					npos2 = follow[j+1].Pos()
				}
				if !p.hasInline(pos2, npos2, nline2) {
					break
				}
				if l := p.stmtCols(s2); l > inlineIndent {
					inlineIndent = l
				}
				if ind2++; ind2 >= len(p.lines) {
					nline2 = maxPos
				} else {
					nline2 = p.lines[ind2]
				}
			}
			if ind2 == p.nlineIndex+1 {
				// no inline comments directly after this one
				continue
			}
		}
		if inlineIndent > 0 {
			if l := p.stmtCols(s); l > 0 {
				p.commentPadding = inlineIndent - l
			}
		}
	}
	p.wantNewline = true
}

type byteCounter int

func (c *byteCounter) WriteByte(b byte) error {
	switch {
	case *c < 0:
	case b == '\n':
		*c = -1
	default:
		*c++
	}
	return nil
}
func (c *byteCounter) WriteString(s string) (int, error) {
	switch {
	case *c < 0:
	case strings.Contains(s, "\n"):
		*c = -1
	default:
		*c += byteCounter(len(s))
	}
	return 0, nil
}
func (c *byteCounter) Reset(io.Writer) { *c = 0 }
func (c *byteCounter) Flush() error    { return nil }

// stmtCols reports the length that s will take when formatted in a
// single line. If it will span multiple lines, stmtCols will return -1.
func (p *Printer) stmtCols(s *Stmt) int {
	*p.lenPrinter = Printer{
		bufWriter: &p.lenCounter,
		lines:     p.lines,
	}
	p.lenPrinter.bufWriter.Reset(nil)
	p.lenPrinter.incLines(s.Pos())
	p.lenPrinter.stmt(s)
	return int(p.lenCounter)
}

func (p *Printer) nestedStmts(stmts []*Stmt, closing Pos) {
	p.incLevel()
	if len(stmts) == 1 && closing > p.nline && stmts[0].End() <= p.nline {
		p.newline(0)
		p.indent()
	}
	p.stmts(stmts)
	p.decLevel()
}

func (p *Printer) assigns(assigns []*Assign, alwaysEqual bool) {
	anyNewline := false
	for _, a := range assigns {
		if a.Pos() > p.nline {
			p.bslashNewl()
			if !anyNewline {
				p.incLevel()
				anyNewline = true
			}
			p.indent()
		} else if p.wantSpace {
			p.WriteByte(' ')
		}
		if a.Name != nil {
			p.WriteString(a.Name.Value)
			p.wroteIndex(a.Index, a.Key)
			if a.Append {
				p.WriteByte('+')
			}
			if alwaysEqual || a.Value != nil || a.Array != nil {
				p.WriteByte('=')
			}
		}
		if a.Value != nil {
			p.word(a.Value)
		} else if a.Array != nil {
			p.wantSpace = false
			p.WriteByte('(')
			p.elemJoin(a.Array.Elems)
			p.sepTok(")", a.Array.Rparen)
		}
		p.wantSpace = true
	}
	if anyNewline {
		p.decLevel()
	}
}
