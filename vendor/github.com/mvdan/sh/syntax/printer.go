// Copyright (c) 2016, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package syntax

import (
	"bufio"
	"io"
	"sync"
)

// PrintConfig controls how the printing of an AST node will behave.
type PrintConfig struct {
	Spaces int // 0 (default) for tabs, >0 for number of spaces
}

var printerFree = sync.Pool{
	New: func() interface{} {
		return &printer{
			bufWriter:  bufio.NewWriter(nil),
			lenPrinter: new(printer),
		}
	},
}

// Fprint "pretty-prints" the given AST file to the given writer.
func (c PrintConfig) Fprint(w io.Writer, f *File) error {
	p := printerFree.Get().(*printer)
	p.reset()
	p.PrintConfig = c
	p.lines, p.comments = f.lines, f.Comments
	p.bufWriter.Reset(w)
	p.stmts(f.Stmts)
	p.commentsUpTo(0)
	p.newline(0)
	var err error
	if flusher, ok := p.bufWriter.(interface {
		Flush() error
	}); ok {
		err = flusher.Flush()
	}
	printerFree.Put(p)
	return err
}

// Fprint "pretty-prints" the given AST file to the given writer. It
// calls PrintConfig.Fprint with its default settings.
func Fprint(w io.Writer, f *File) error {
	return PrintConfig{}.Fprint(w, f)
}

type bufWriter interface {
	WriteByte(byte) error
	WriteString(string) (int, error)
	Reset(io.Writer)
}

type printer struct {
	bufWriter

	PrintConfig
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

	// used in stmtLen to align comments
	lenPrinter *printer
	lenCounter byteCounter
}

func (p *printer) reset() {
	p.wantSpace, p.wantNewline = false, false
	p.commentPadding = 0
	p.nline, p.nlineIndex = 0, 0
	p.lastLevel, p.level = 0, 0
	p.levelIncs = p.levelIncs[:0]
	p.nestedBinary = false
	p.pendingHdocs = p.pendingHdocs[:0]
}

func (p *printer) incLine() {
	if p.nlineIndex++; p.nlineIndex >= len(p.lines) {
		p.nline = maxPos
	} else {
		p.nline = p.lines[p.nlineIndex]
	}
}

func (p *printer) incLines(pos Pos) {
	for p.nline < pos {
		p.incLine()
	}
}

func (p *printer) spaces(n int) {
	for i := 0; i < n; i++ {
		p.WriteByte(' ')
	}
}

func (p *printer) bslashNewl() {
	if p.wantSpace {
		p.WriteByte(' ')
	}
	p.WriteString("\\\n")
	p.wantSpace = false
	p.incLine()
}

func (p *printer) spacedString(s string) {
	if p.wantSpace {
		p.WriteByte(' ')
	}
	p.WriteString(s)
	p.wantSpace = true
}

func (p *printer) semiOrNewl(s string, pos Pos) {
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

func (p *printer) incLevel() {
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

func (p *printer) decLevel() {
	if p.levelIncs[len(p.levelIncs)-1] {
		p.level--
	}
	p.levelIncs = p.levelIncs[:len(p.levelIncs)-1]
}

func (p *printer) indent() {
	p.lastLevel = p.level
	switch {
	case p.level == 0:
	case p.Spaces == 0:
		for i := 0; i < p.level; i++ {
			p.WriteByte('\t')
		}
	case p.Spaces > 0:
		p.spaces(p.Spaces * p.level)
	}
}

func (p *printer) newline(pos Pos) {
	p.wantNewline, p.wantSpace = false, false
	p.WriteByte('\n')
	if pos > p.nline {
		p.incLine()
	}
	hdocs := p.pendingHdocs
	p.pendingHdocs = p.pendingHdocs[:0]
	for _, r := range hdocs {
		p.word(r.Hdoc)
		p.incLines(r.Hdoc.End())
		p.unquotedWord(r.Word)
		p.WriteByte('\n')
		p.incLine()
		p.wantSpace = false
	}
}

func (p *printer) newlines(pos Pos) {
	p.newline(pos)
	if pos > p.nline {
		// preserve single empty lines
		p.WriteByte('\n')
		p.incLine()
	}
	p.indent()
}

func (p *printer) commentsAndSeparate(pos Pos) {
	p.commentsUpTo(pos)
	if p.wantNewline || pos > p.nline {
		p.newlines(pos)
	}
}

func (p *printer) sepTok(s string, pos Pos) {
	p.level++
	p.commentsUpTo(pos)
	p.level--
	if p.wantNewline || pos > p.nline {
		p.newlines(pos)
	}
	p.WriteString(s)
	p.wantSpace = true
}

func (p *printer) semiRsrv(s string, pos Pos, fallback bool) {
	p.level++
	p.commentsUpTo(pos)
	p.level--
	if p.wantNewline || pos > p.nline {
		p.newlines(pos)
	} else if fallback {
		if !p.wroteSemi {
			p.WriteByte(';')
		}
		p.WriteByte(' ')
	} else if p.wantSpace {
		p.WriteByte(' ')
	}
	p.WriteString(s)
	p.wantSpace = true
}

func (p *printer) anyCommentsBefore(pos Pos) bool {
	if !pos.IsValid() || len(p.comments) < 1 {
		return false
	}
	return p.comments[0].Hash < pos
}

func (p *printer) commentsUpTo(pos Pos) {
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

func (p *printer) wordPart(wp WordPart) {
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
		if x.Dollar {
			p.WriteByte('$')
		}
		p.WriteByte('"')
		for i, n := range x.Parts {
			p.wordPart(n)
			if i == len(x.Parts)-1 {
				p.incLines(n.End())
			}
		}
		p.WriteByte('"')
	case *CmdSubst:
		p.incLines(x.Pos())
		p.WriteString("$(")
		p.wantSpace = len(x.Stmts) > 0 && startsWithLparen(x.Stmts[0])
		p.nestedStmts(x.Stmts, x.Right)
		p.sepTok(")", x.Right)
	case *ParamExp:
		if x.Short {
			p.WriteByte('$')
			p.WriteString(x.Param.Value)
			break
		}
		p.WriteString("${")
		switch {
		case x.Length:
			p.WriteByte('#')
		case x.Excl:
			p.WriteByte('!')
		}
		if x.Param != nil {
			p.WriteString(x.Param.Value)
		}
		if x.Ind != nil {
			p.WriteByte('[')
			p.arithmExpr(x.Ind.Expr, false)
			p.WriteByte(']')
		}
		if x.Slice != nil {
			p.WriteByte(':')
			if un, ok := x.Slice.Offset.(*UnaryArithm); ok {
				if un.Op == Plus || un.Op == Minus {
					// to avoid :+ and :-
					p.WriteByte(' ')
				}
			}
			p.arithmExpr(x.Slice.Offset, true)
			if x.Slice.Length != nil {
				p.WriteByte(':')
				p.arithmExpr(x.Slice.Length, true)
			}
		} else if x.Repl != nil {
			if x.Repl.All {
				p.WriteByte('/')
			}
			p.WriteByte('/')
			p.word(x.Repl.Orig)
			p.WriteByte('/')
			p.word(x.Repl.With)
		} else if x.Exp != nil {
			p.WriteString(x.Exp.Op.String())
			p.word(x.Exp.Word)
		}
		p.WriteByte('}')
	case *ArithmExp:
		p.WriteString("$((")
		p.arithmExpr(x.X, false)
		p.WriteString("))")
	case *ArrayExpr:
		p.wantSpace = false
		p.WriteByte('(')
		p.wordJoin(x.List, false)
		p.sepTok(")", x.Rparen)
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

func (p *printer) loop(loop Loop) {
	switch x := loop.(type) {
	case *WordIter:
		p.WriteString(x.Name.Value)
		if len(x.List) > 0 {
			p.spacedString(" in")
			p.wordJoin(x.List, true)
		}
	case *CStyleLoop:
		p.WriteString("((")
		if x.Init == nil {
			p.WriteByte(' ')
		}
		p.arithmExpr(x.Init, false)
		p.WriteString("; ")
		p.arithmExpr(x.Cond, false)
		p.WriteString("; ")
		p.arithmExpr(x.Post, false)
		p.WriteString("))")
	}
}

func (p *printer) arithmExpr(expr ArithmExpr, compact bool) {
	switch x := expr.(type) {
	case *Word:
		p.word(x)
	case *BinaryArithm:
		if compact {
			p.arithmExpr(x.X, compact)
			p.WriteString(x.Op.String())
			p.arithmExpr(x.Y, compact)
		} else {
			p.arithmExpr(x.X, compact)
			if x.Op != Comma {
				p.WriteByte(' ')
			}
			p.WriteString(x.Op.String())
			p.WriteByte(' ')
			p.arithmExpr(x.Y, compact)
		}
	case *UnaryArithm:
		if x.Post {
			p.arithmExpr(x.X, compact)
			p.WriteString(x.Op.String())
		} else {
			p.WriteString(x.Op.String())
			p.arithmExpr(x.X, compact)
		}
	case *ParenArithm:
		p.WriteByte('(')
		p.arithmExpr(x.X, false)
		p.WriteByte(')')
	}
}

func (p *printer) testExpr(expr TestExpr) {
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

func (p *printer) word(w *Word) {
	for _, n := range w.Parts {
		p.wordPart(n)
	}
	p.wantSpace = true
}

func (p *printer) unquotedWord(w *Word) {
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

func (p *printer) wordJoin(ws []*Word, backslash bool) {
	anyNewline := false
	for _, w := range ws {
		if pos := w.Pos(); pos > p.nline {
			p.commentsUpTo(pos)
			if backslash {
				p.bslashNewl()
			} else {
				p.WriteByte('\n')
				p.incLine()
			}
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

func (p *printer) stmt(s *Stmt) {
	if s.Negated {
		p.spacedString("!")
	}
	p.assigns(s.Assigns)
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
		p.word(r.Word)
		if r.Op == Hdoc || r.Op == DashHdoc {
			p.pendingHdocs = append(p.pendingHdocs, r)
		}
	}
	p.wroteSemi = false
	if s.Semicolon.IsValid() && s.Semicolon > p.nline {
		p.incLevel()
		p.bslashNewl()
		p.indent()
		p.decLevel()
		p.WriteByte(';')
		p.wroteSemi = true
	} else if s.Background {
		p.WriteString(" &")
	}
	if anyNewline {
		p.decLevel()
	}
}

func (p *printer) command(cmd Command, redirs []*Redirect) (startRedirs int) {
	if p.wantSpace {
		p.WriteByte(' ')
		p.wantSpace = false
	}
	switch x := cmd.(type) {
	case *CallExpr:
		if len(x.Args) <= 1 {
			p.wordJoin(x.Args, true)
			return 0
		}
		p.wordJoin(x.Args[:1], true)
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
			p.word(r.Word)
			startRedirs++
		}
		p.wordJoin(x.Args[1:], true)
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
		p.spacedString("while")
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
		indent := !p.nestedBinary
		if indent {
			p.incLevel()
		}
		_, p.nestedBinary = x.Y.Cmd.(*BinaryCmd)
		if len(p.pendingHdocs) == 0 && x.Y.Pos() > p.nline {
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
		p.incLines(x.Y.Pos())
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
		p.incLevel()
		for _, pl := range x.List {
			p.commentsAndSeparate(pl.Patterns[0].Pos())
			for i, w := range pl.Patterns {
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
			sep := len(pl.Stmts) > 1 || (len(pl.Stmts) > 0 && pl.Stmts[0].Pos() > p.nline)
			p.nestedStmts(pl.Stmts, 0)
			p.level++
			if sep {
				p.commentsUpTo(pl.OpPos)
				p.newlines(pl.OpPos)
			}
			p.spacedString(pl.Op.String())
			p.incLines(pl.OpPos)
			p.level--
			if sep || pl.OpPos == x.Esac {
				p.wantNewline = true
			}
		}
		p.decLevel()
		p.semiRsrv("esac", x.Esac, len(x.List) == 0)
	case *UntilClause:
		p.spacedString("until")
		p.nestedStmts(x.CondStmts, 0)
		p.semiOrNewl("do", x.Do)
		p.nestedStmts(x.DoStmts, 0)
		p.semiRsrv("done", x.Done, true)
	case *ArithmCmd:
		p.WriteString("((")
		p.arithmExpr(x.X, false)
		p.WriteString("))")
	case *TestClause:
		p.WriteString("[[ ")
		p.testExpr(x.X)
		p.spacedString("]]")
	case *DeclClause:
		name := x.Variant
		if name == "" {
			name = "declare"
		}
		p.spacedString(name)
		for _, w := range x.Opts {
			p.WriteByte(' ')
			p.word(w)
		}
		p.assigns(x.Assigns)
	case *EvalClause:
		p.spacedString("eval")
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
			p.arithmExpr(n, true)
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

func (p *printer) hasInline(pos, npos, nline Pos) bool {
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

func (p *printer) stmts(stmts []*Stmt) {
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
				if pos2 > nline2 || !p.hasInline(pos2, npos2, nline2) {
					break
				}
				if l := p.stmtLen(s2); l > inlineIndent {
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
			p.commentPadding = inlineIndent - p.stmtLen(s)
		}
	}
	p.wantNewline = true
}

type byteCounter int

func (c *byteCounter) WriteByte(b byte) error {
	*c++
	return nil
}
func (c *byteCounter) WriteString(s string) (int, error) {
	*c += byteCounter(len(s))
	return 0, nil
}
func (c *byteCounter) Reset(io.Writer) { *c = 0 }

func (p *printer) stmtLen(s *Stmt) int {
	*p.lenPrinter = printer{bufWriter: &p.lenCounter}
	p.lenPrinter.bufWriter.Reset(nil)
	p.lenPrinter.incLines(s.Pos())
	p.lenPrinter.stmt(s)
	return int(p.lenCounter)
}

func (p *printer) nestedStmts(stmts []*Stmt, closing Pos) {
	p.incLevel()
	if len(stmts) == 1 && closing > p.nline && stmts[0].End() <= p.nline {
		p.newline(0)
		p.indent()
	}
	p.stmts(stmts)
	p.decLevel()
}

func (p *printer) assigns(assigns []*Assign) {
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
			if a.Append {
				p.WriteByte('+')
			}
			p.WriteByte('=')
		}
		if a.Value != nil {
			p.word(a.Value)
		}
		p.wantSpace = true
	}
	if anyNewline {
		p.decLevel()
	}
}
