// Copyright (c) 2016, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package syntax

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"unicode"
)

// PrinterOption is a function which can be passed to NewPrinter
// to alter its behaviour. To apply option to existing Printer
// call it directly, for example KeepPadding(true)(printer).
type PrinterOption func(*Printer)

// Indent sets the number of spaces used for indentation. If set to 0,
// tabs will be used instead.
func Indent(spaces uint) PrinterOption {
	return func(p *Printer) { p.indentSpaces = spaces }
}

// BinaryNextLine will make binary operators appear on the next line
// when a binary command, such as a pipe, spans multiple lines. A
// backslash will be used.
func BinaryNextLine(enabled bool) PrinterOption {
	return func(p *Printer) { p.binNextLine = enabled }
}

// SwitchCaseIndent will make switch cases be indented. As such, switch
// case bodies will be two levels deeper than the switch itself.
func SwitchCaseIndent(enabled bool) PrinterOption {
	return func(p *Printer) { p.swtCaseIndent = enabled }
}

// TODO(v4): consider turning this into a "space all operators" option, to also
// allow foo=( bar baz ), (( x + y )), and so on.

// SpaceRedirects will put a space after most redirection operators. The
// exceptions are '>&', '<&', '>(', and '<('.
func SpaceRedirects(enabled bool) PrinterOption {
	return func(p *Printer) { p.spaceRedirects = enabled }
}

// KeepPadding will keep most nodes and tokens in the same column that
// they were in the original source. This allows the user to decide how
// to align and pad their code with spaces.
//
// Note that this feature is best-effort and will only keep the
// alignment stable, so it may need some human help the first time it is
// run.
func KeepPadding(enabled bool) PrinterOption {
	return func(p *Printer) {
		if enabled && !p.keepPadding {
			// Enable the flag, and set up the writer wrapper.
			p.keepPadding = true
			p.cols.Writer = p.bufWriter.(*bufio.Writer)
			p.bufWriter = &p.cols

		} else if !enabled && p.keepPadding {
			// Ensure we reset the state to that of NewPrinter.
			p.keepPadding = false
			p.bufWriter = p.cols.Writer
			p.cols = colCounter{}
		}
	}
}

// Minify will print programs in a way to save the most bytes possible.
// For example, indentation and comments are skipped, and extra
// whitespace is avoided when possible.
func Minify(enabled bool) PrinterOption {
	return func(p *Printer) { p.minify = enabled }
}

// FunctionNextLine will place a function's opening braces on the next line.
func FunctionNextLine(enabled bool) PrinterOption {
	return func(p *Printer) { p.funcNextLine = enabled }
}

// NewPrinter allocates a new Printer and applies any number of options.
func NewPrinter(opts ...PrinterOption) *Printer {
	p := &Printer{
		bufWriter: bufio.NewWriter(nil),
		tabWriter: new(tabwriter.Writer),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Print "pretty-prints" the given syntax tree node to the given writer. Writes
// to w are buffered.
//
// The node types supported at the moment are *File, *Stmt, *Word, any Command
// node, and any WordPart node. A trailing newline will only be printed when a
// *File is used.
func (p *Printer) Print(w io.Writer, node Node) error {
	p.reset()

	// TODO: consider adding a raw mode to skip the tab writer, much like in
	// go/printer.
	twmode := tabwriter.DiscardEmptyColumns | tabwriter.StripEscape
	tabwidth := 8
	if p.indentSpaces == 0 {
		// indenting with tabs
		twmode |= tabwriter.TabIndent
	} else {
		// indenting with spaces
		tabwidth = int(p.indentSpaces)
	}
	p.tabWriter.Init(w, 0, tabwidth, 1, ' ', twmode)
	w = p.tabWriter

	p.bufWriter.Reset(w)
	switch x := node.(type) {
	case *File:
		p.stmtList(x.Stmts, x.Last)
		p.newline(x.End())
	case *Stmt:
		p.stmtList([]*Stmt{x}, nil)
	case Command:
		p.line = x.Pos().Line()
		p.command(x, nil)
	case *Word:
		p.line = x.Pos().Line()
		p.word(x)
	case WordPart:
		p.line = x.Pos().Line()
		p.wordPart(x, nil)
	default:
		return fmt.Errorf("unsupported node type: %T", x)
	}
	p.flushHeredocs()
	p.flushComments()

	// flush the writers
	if err := p.bufWriter.Flush(); err != nil {
		return err
	}
	if tw, _ := w.(*tabwriter.Writer); tw != nil {
		if err := tw.Flush(); err != nil {
			return err
		}
	}
	return nil
}

type bufWriter interface {
	Write([]byte) (int, error)
	WriteString(string) (int, error)
	WriteByte(byte) error
	Reset(io.Writer)
	Flush() error
}

type colCounter struct {
	*bufio.Writer
	column    int
	lineStart bool
}

func (c *colCounter) WriteByte(b byte) error {
	switch b {
	case '\n':
		c.column = 0
		c.lineStart = true
	case '\t', ' ':
	default:
		c.lineStart = false
	}
	c.column++
	return c.Writer.WriteByte(b)
}

func (c *colCounter) WriteString(s string) (int, error) {
	c.lineStart = false
	for _, r := range s {
		if r == '\n' {
			c.column = 0
		}
		c.column++
	}
	return c.Writer.WriteString(s)
}

func (c *colCounter) Reset(w io.Writer) {
	c.column = 1
	c.lineStart = true
	c.Writer.Reset(w)
}

// Printer holds the internal state of the printing mechanism of a
// program.
type Printer struct {
	bufWriter
	tabWriter *tabwriter.Writer
	cols      colCounter

	indentSpaces   uint
	binNextLine    bool
	swtCaseIndent  bool
	spaceRedirects bool
	keepPadding    bool
	minify         bool
	funcNextLine   bool

	wantSpace   bool
	wantNewline bool
	wroteSemi   bool

	// pendingComments are any comments in the current line or statement
	// that we have yet to print. This is useful because that way, we can
	// ensure that all comments are written immediately before a newline.
	// Otherwise, in some edge cases we might wrongly place words after a
	// comment in the same line, breaking programs.
	pendingComments []Comment

	// firstLine means we are still writing the first line
	firstLine bool
	// line is the current line number
	line uint

	// lastLevel is the last level of indentation that was used.
	lastLevel uint
	// level is the current level of indentation.
	level uint
	// levelIncs records which indentation level increments actually
	// took place, to revert them once their section ends.
	levelIncs []bool

	nestedBinary bool

	// pendingHdocs is the list of pending heredocs to write.
	pendingHdocs []*Redirect

	// used when printing <<- heredocs with tab indentation
	tabsPrinter *Printer
}

func (p *Printer) reset() {
	p.wantSpace, p.wantNewline = false, false
	p.pendingComments = p.pendingComments[:0]

	// minification uses its own newline logic
	p.firstLine = !p.minify
	p.line = 0

	p.lastLevel, p.level = 0, 0
	p.levelIncs = p.levelIncs[:0]
	p.nestedBinary = false
	p.pendingHdocs = p.pendingHdocs[:0]
}

func (p *Printer) spaces(n uint) {
	for i := uint(0); i < n; i++ {
		p.WriteByte(' ')
	}
}

func (p *Printer) space() {
	p.WriteByte(' ')
	p.wantSpace = false
}

func (p *Printer) spacePad(pos Pos) {
	if p.wantSpace {
		p.WriteByte(' ')
		p.wantSpace = false
	}
	if p.cols.lineStart {
		// Never add padding at the start of a line, since this may
		// result in broken indentation or mixing of spaces and tabs.
		return
	}
	for !p.cols.lineStart && p.cols.column > 0 && p.cols.column < int(pos.col) {
		p.WriteByte(' ')
	}
}

func (p *Printer) bslashNewl() {
	if p.wantSpace {
		p.space()
	}
	p.WriteString("\\\n")
	p.line++
	p.indent()
}

func (p *Printer) spacedString(s string, pos Pos) {
	p.spacePad(pos)
	p.WriteString(s)
	p.wantSpace = true
}

func (p *Printer) spacedToken(s string, pos Pos) {
	if p.minify {
		p.WriteString(s)
		p.wantSpace = false
		return
	}
	p.spacePad(pos)
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
		if !p.minify {
			p.space()
		}
		p.line = pos.Line()
	}
	p.WriteString(s)
	p.wantSpace = true
}

func (p *Printer) writeLit(s string) {
	if !strings.Contains(s, "\t") {
		p.WriteString(s)
		return
	}
	p.WriteByte('\xff')
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b != '\t' {
			p.WriteByte(b)
			continue
		}
		p.WriteByte(b)
	}
	p.WriteByte('\xff')
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
	if p.minify {
		return
	}
	p.lastLevel = p.level
	switch {
	case p.level == 0:
	case p.indentSpaces == 0:
		p.WriteByte('\xff')
		for i := uint(0); i < p.level; i++ {
			p.WriteByte('\t')
		}
		p.WriteByte('\xff')
	default:
		p.spaces(p.indentSpaces * p.level)
	}
}

func (p *Printer) newline(pos Pos) {
	p.flushHeredocs()
	p.flushComments()
	p.WriteByte('\n')
	p.wantNewline, p.wantSpace = false, false
	if p.line < pos.Line() {
		p.line++
	}
}

func (p *Printer) flushHeredocs() {
	if len(p.pendingHdocs) == 0 {
		return
	}
	hdocs := p.pendingHdocs
	p.pendingHdocs = p.pendingHdocs[:0]
	coms := p.pendingComments
	p.pendingComments = nil
	if len(coms) > 0 {
		c := coms[0]
		if c.Pos().Line() == p.line {
			p.pendingComments = append(p.pendingComments, c)
			p.flushComments()
			coms = coms[1:]
		}
	}

	// Reuse the last indentation level, as
	// indentation levels are usually changed before
	// newlines are printed along with their
	// subsequent indentation characters.
	newLevel := p.level
	p.level = p.lastLevel

	for _, r := range hdocs {
		p.line++
		p.WriteByte('\n')
		p.wantNewline, p.wantSpace = false, false
		if r.Op == DashHdoc && p.indentSpaces == 0 && !p.minify {
			if r.Hdoc != nil {
				extra := extraIndenter{
					bufWriter:   p.bufWriter,
					baseIndent:  int(p.level + 1),
					firstIndent: -1,
				}
				p.tabsPrinter = &Printer{
					bufWriter: &extra,
					line:      r.Hdoc.Pos().Line(),
				}
				p.tabsPrinter.wordParts(r.Hdoc.Parts, true)
			}
			p.indent()
		} else if r.Hdoc != nil {
			p.wordParts(r.Hdoc.Parts, true)
		}
		p.unquotedWord(r.Word)
		if r.Hdoc != nil {
			// Overwrite p.line, since printing r.Word again can set
			// p.line to the beginning of the heredoc again.
			p.line = r.Hdoc.End().Line()
		}
		p.wantSpace = false
	}
	p.level = newLevel
	p.pendingComments = coms
}

func (p *Printer) newlines(pos Pos) {
	if p.firstLine && len(p.pendingComments) == 0 {
		p.firstLine = false
		return // no empty lines at the top
	}
	if !p.wantNewline && pos.Line() <= p.line {
		return
	}
	p.newline(pos)
	if pos.Line() > p.line {
		if !p.minify {
			// preserve single empty lines
			p.WriteByte('\n')
		}
		p.line++
	}
	p.indent()
}

func (p *Printer) rightParen(pos Pos) {
	if !p.minify {
		p.newlines(pos)
	}
	p.WriteByte(')')
	p.wantSpace = true
}

func (p *Printer) semiRsrv(s string, pos Pos) {
	if p.wantNewline || pos.Line() > p.line {
		p.newlines(pos)
	} else {
		if !p.wroteSemi {
			p.WriteByte(';')
		}
		if !p.minify {
			p.spacePad(pos)
		}
	}
	p.WriteString(s)
	p.wantSpace = true
}

func (p *Printer) flushComments() {
	for i, c := range p.pendingComments {
		if i == 0 {
			// Flush any pending heredocs first. Otherwise, the
			// comments would become part of a heredoc body.
			p.flushHeredocs()
		}
		p.firstLine = false
		// We can't call any of the newline methods, as they call this
		// function and we'd recurse forever.
		cline := c.Hash.Line()
		switch {
		case i > 0, cline > p.line && p.line > 0:
			p.WriteByte('\n')
			if cline > p.line+1 {
				p.WriteByte('\n')
			}
			p.indent()
		case p.wantSpace:
			if p.keepPadding {
				p.spacePad(c.Pos())
			} else {
				p.WriteByte('\t')
			}
		}
		// don't go back one line, which may happen in some edge cases
		if p.line < cline {
			p.line = cline
		}
		p.WriteByte('#')
		p.writeLit(strings.TrimRightFunc(c.Text, unicode.IsSpace))
		p.wantNewline = true
	}
	p.pendingComments = nil
}

func (p *Printer) comments(comments ...Comment) {
	if p.minify {
		return
	}
	p.pendingComments = append(p.pendingComments, comments...)
}

func (p *Printer) wordParts(wps []WordPart, quoted bool) {
	for i, wp := range wps {
		var next WordPart
		if i+1 < len(wps) {
			next = wps[i+1]
		}
		for wp.Pos().Line() > p.line {
			if quoted {
				// No extra spacing or indentation if quoted.
				p.WriteString("\\\n")
				p.line++
			} else {
				p.bslashNewl()
			}
		}
		p.wordPart(wp, next)
		p.line = wp.End().Line()
	}
}

func (p *Printer) wordPart(wp, next WordPart) {
	switch x := wp.(type) {
	case *Lit:
		p.writeLit(x.Value)
	case *SglQuoted:
		if x.Dollar {
			p.WriteByte('$')
		}
		p.WriteByte('\'')
		p.writeLit(x.Value)
		p.WriteByte('\'')
		p.line = x.End().Line()
	case *DblQuoted:
		p.dblQuoted(x)
	case *CmdSubst:
		p.line = x.Pos().Line()
		switch {
		case x.TempFile:
			p.WriteString("${")
			p.wantSpace = true
			p.nestedStmts(x.Stmts, x.Last, x.Right)
			p.wantSpace = false
			p.semiRsrv("}", x.Right)
		case x.ReplyVar:
			p.WriteString("${|")
			p.nestedStmts(x.Stmts, x.Last, x.Right)
			p.wantSpace = false
			p.semiRsrv("}", x.Right)
		default:
			p.WriteString("$(")
			p.wantSpace = len(x.Stmts) > 0 && startsWithLparen(x.Stmts[0])
			p.nestedStmts(x.Stmts, x.Last, x.Right)
			p.rightParen(x.Right)
		}
	case *ParamExp:
		litCont := ";"
		if nextLit, ok := next.(*Lit); ok && nextLit.Value != "" {
			litCont = nextLit.Value[:1]
		}
		name := x.Param.Value
		switch {
		case !p.minify:
		case x.Excl, x.Length, x.Width:
		case x.Index != nil, x.Slice != nil:
		case x.Repl != nil, x.Exp != nil:
		case len(name) > 1 && !ValidName(name): // ${10}
		case ValidName(name + litCont): // ${var}cont
		default:
			x2 := *x
			x2.Short = true
			p.paramExp(&x2)
			return
		}
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
		p.writeLit(x.Pattern.Value)
		p.WriteByte(')')
	case *ProcSubst:
		// avoid conflict with << and others
		if p.wantSpace {
			p.space()
		}
		p.WriteString(x.Op.String())
		p.nestedStmts(x.Stmts, x.Last, x.Rparen)
		p.rightParen(x.Rparen)
	}
}

func (p *Printer) dblQuoted(dq *DblQuoted) {
	if dq.Dollar {
		p.WriteByte('$')
	}
	p.WriteByte('"')
	if len(dq.Parts) > 0 {
		p.wordParts(dq.Parts, true)
	}
	// Add any trailing escaped newlines.
	for p.line < dq.Right.Line() {
		p.WriteString("\\\n")
		p.line++
	}
	p.WriteByte('"')
}

func (p *Printer) wroteIndex(index ArithmExpr) bool {
	if index == nil {
		return false
	}
	p.WriteByte('[')
	p.arithmExpr(index, false, false)
	p.WriteByte(']')
	return true
}

func (p *Printer) paramExp(pe *ParamExp) {
	if pe.nakedIndex() { // arr[x]
		p.writeLit(pe.Param.Value)
		p.wroteIndex(pe.Index)
		return
	}
	if pe.Short { // $var
		p.WriteByte('$')
		p.writeLit(pe.Param.Value)
		return
	}
	// ${var...}
	p.WriteString("${")
	switch {
	case pe.Length:
		p.WriteByte('#')
	case pe.Width:
		p.WriteByte('%')
	case pe.Excl:
		p.WriteByte('!')
	}
	p.writeLit(pe.Param.Value)
	p.wroteIndex(pe.Index)
	switch {
	case pe.Slice != nil:
		p.WriteByte(':')
		p.arithmExpr(pe.Slice.Offset, true, true)
		if pe.Slice.Length != nil {
			p.WriteByte(':')
			p.arithmExpr(pe.Slice.Length, true, false)
		}
	case pe.Repl != nil:
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
	case pe.Names != 0:
		p.writeLit(pe.Names.String())
	case pe.Exp != nil:
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
		p.writeLit(x.Name.Value)
		if x.InPos.IsValid() {
			p.spacedString(" in", Pos{})
			p.wordJoin(x.Items)
		}
	case *CStyleLoop:
		p.WriteString("((")
		if x.Init == nil {
			p.space()
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
	if p.minify {
		compact = true
	}
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
				p.space()
			}
			p.WriteString(x.Op.String())
			p.space()
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
					p.space()
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
		p.space()
		p.WriteString(x.Op.String())
		p.space()
		p.testExpr(x.Y)
	case *UnaryTest:
		p.WriteString(x.Op.String())
		p.space()
		p.testExpr(x.X)
	case *ParenTest:
		p.WriteByte('(')
		p.testExpr(x.X)
		p.WriteByte(')')
	}
}

func (p *Printer) word(w *Word) {
	p.wordParts(w.Parts, false)
	p.wantSpace = true
}

func (p *Printer) unquotedWord(w *Word) {
	for _, wp := range w.Parts {
		switch x := wp.(type) {
		case *SglQuoted:
			p.writeLit(x.Value)
		case *DblQuoted:
			p.wordParts(x.Parts, true)
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
		if pos := w.Pos(); pos.Line() > p.line {
			if !anyNewline {
				p.incLevel()
				anyNewline = true
			}
			p.bslashNewl()
		} else {
			p.spacePad(w.Pos())
		}
		p.word(w)
	}
	if anyNewline {
		p.decLevel()
	}
}

func (p *Printer) casePatternJoin(pats []*Word) {
	anyNewline := false
	for i, w := range pats {
		if i > 0 {
			p.spacedToken("|", Pos{})
		}
		if pos := w.Pos(); pos.Line() > p.line {
			if !anyNewline {
				p.incLevel()
				anyNewline = true
			}
			p.bslashNewl()
		} else {
			p.spacePad(w.Pos())
		}
		p.word(w)
	}
	if anyNewline {
		p.decLevel()
	}
}

func (p *Printer) elemJoin(elems []*ArrayElem, last []Comment) {
	p.incLevel()
	for _, el := range elems {
		var left []Comment
		for _, c := range el.Comments {
			if c.Pos().After(el.Pos()) {
				left = append(left, c)
				break
			}
			p.comments(c)
		}
		if el.Pos().Line() > p.line {
			p.newlines(el.Pos())
		} else if p.wantSpace {
			p.space()
		}
		if p.wroteIndex(el.Index) {
			p.WriteByte('=')
		}
		if el.Value != nil {
			p.word(el.Value)
		}
		p.comments(left...)
	}
	if len(last) > 0 {
		p.comments(last...)
		p.flushComments()
	}
	p.decLevel()
}

func (p *Printer) stmt(s *Stmt) {
	p.wroteSemi = false
	if s.Negated {
		p.spacedString("!", s.Pos())
	}
	var startRedirs int
	if s.Cmd != nil {
		startRedirs = p.command(s.Cmd, s.Redirs)
	}
	p.incLevel()
	for _, r := range s.Redirs[startRedirs:] {
		if r.OpPos.Line() > p.line {
			p.bslashNewl()
		}
		if p.wantSpace {
			p.spacePad(r.Pos())
		}
		if r.N != nil {
			p.writeLit(r.N.Value)
		}
		p.WriteString(r.Op.String())
		if p.spaceRedirects && (r.Op != DplIn && r.Op != DplOut) {
			p.space()
		} else {
			p.wantSpace = true
		}
		p.word(r.Word)
		if r.Op == Hdoc || r.Op == DashHdoc {
			p.pendingHdocs = append(p.pendingHdocs, r)
		}
	}
	p.wroteSemi = true
	switch {
	case s.Semicolon.IsValid() && s.Semicolon.Line() > p.line:
		p.bslashNewl()
		p.WriteByte(';')
	case s.Background:
		if !p.minify {
			p.space()
		}
		p.WriteString("&")
	case s.Coprocess:
		if !p.minify {
			p.space()
		}
		p.WriteString("|&")
	default:
		p.wroteSemi = false
	}
	if p.wroteSemi {
		p.wantSpace = true
	}
	p.decLevel()
}

func (p *Printer) command(cmd Command, redirs []*Redirect) (startRedirs int) {
	p.spacePad(cmd.Pos())
	switch x := cmd.(type) {
	case *CallExpr:
		p.assigns(x.Assigns)
		if len(x.Args) <= 1 {
			p.wordJoin(x.Args)
			return 0
		}
		p.wordJoin(x.Args[:1])
		for _, r := range redirs {
			if r.Pos().After(x.Args[1].Pos()) || r.Op == Hdoc || r.Op == DashHdoc {
				break
			}
			if p.wantSpace {
				p.spacePad(r.Pos())
			}
			if r.N != nil {
				p.writeLit(r.N.Value)
			}
			p.WriteString(r.Op.String())
			if p.spaceRedirects && (r.Op != DplIn && r.Op != DplOut) {
				p.space()
			} else {
				p.wantSpace = true
			}
			p.word(r.Word)
			startRedirs++
		}
		p.wordJoin(x.Args[1:])
	case *Block:
		p.WriteByte('{')
		p.wantSpace = true
		// Forbid "foo()\n{ bar; }"
		p.wantNewline = p.wantNewline || p.funcNextLine
		p.nestedStmts(x.Stmts, x.Last, x.Rbrace)
		p.semiRsrv("}", x.Rbrace)
	case *IfClause:
		p.ifClause(x, false)
	case *Subshell:
		p.WriteByte('(')
		p.wantSpace = len(x.Stmts) > 0 && startsWithLparen(x.Stmts[0])
		p.spacePad(stmtsPos(x.Stmts, x.Last))
		p.nestedStmts(x.Stmts, x.Last, x.Rparen)
		p.wantSpace = false
		p.spacePad(x.Rparen)
		p.rightParen(x.Rparen)
	case *WhileClause:
		if x.Until {
			p.spacedString("until", x.Pos())
		} else {
			p.spacedString("while", x.Pos())
		}
		p.nestedStmts(x.Cond, x.CondLast, Pos{})
		p.semiOrNewl("do", x.DoPos)
		p.nestedStmts(x.Do, x.DoLast, x.DonePos)
		p.semiRsrv("done", x.DonePos)
	case *ForClause:
		if x.Select {
			p.WriteString("select ")
		} else {
			p.WriteString("for ")
		}
		p.loop(x.Loop)
		p.semiOrNewl("do", x.DoPos)
		p.nestedStmts(x.Do, x.DoLast, x.DonePos)
		p.semiRsrv("done", x.DonePos)
	case *BinaryCmd:
		p.stmt(x.X)
		if p.minify || x.Y.Pos().Line() <= p.line {
			// leave p.nestedBinary untouched
			p.spacedToken(x.Op.String(), x.OpPos)
			p.line = x.Y.Pos().Line()
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
			}
			p.spacedToken(x.Op.String(), x.OpPos)
			if len(x.Y.Comments) > 0 {
				p.wantSpace = false
				p.newline(Pos{})
				p.indent()
				p.comments(x.Y.Comments...)
				p.newline(Pos{})
				p.indent()
			}
		} else {
			p.spacedToken(x.Op.String(), x.OpPos)
			p.line = x.OpPos.Line()
			p.comments(x.Y.Comments...)
			p.newline(Pos{})
			p.indent()
		}
		p.line = x.Y.Pos().Line()
		_, p.nestedBinary = x.Y.Cmd.(*BinaryCmd)
		p.stmt(x.Y)
		if indent {
			p.decLevel()
		}
		p.nestedBinary = false
	case *FuncDecl:
		if x.RsrvWord {
			p.WriteString("function ")
		}
		p.writeLit(x.Name.Value)
		p.WriteString("()")
		if p.funcNextLine {
			p.newline(Pos{})
		} else if !p.minify {
			p.space()
		}
		p.line = x.Body.Pos().Line()
		p.comments(x.Body.Comments...)
		p.stmt(x.Body)
	case *CaseClause:
		p.WriteString("case ")
		p.word(x.Word)
		p.WriteString(" in")
		if p.swtCaseIndent {
			p.incLevel()
		}
		for i, ci := range x.Items {
			var last []Comment
			for i, c := range ci.Comments {
				if c.Pos().After(ci.Pos()) {
					last = ci.Comments[i:]
					break
				}
				p.comments(c)
			}
			p.newlines(ci.Pos())
			p.casePatternJoin(ci.Patterns)
			p.WriteByte(')')
			p.wantSpace = !p.minify

			bodyPos := stmtsPos(ci.Stmts, ci.Last)
			bodyEnd := stmtsEnd(ci.Stmts, ci.Last)
			sep := len(ci.Stmts) > 1 || bodyPos.Line() > p.line ||
				(bodyEnd.IsValid() && ci.OpPos.Line() > bodyEnd.Line())
			p.nestedStmts(ci.Stmts, ci.Last, ci.OpPos)
			p.level++
			if !p.minify || i != len(x.Items)-1 {
				if sep {
					p.newlines(ci.OpPos)
					p.wantNewline = true
				}
				p.spacedToken(ci.Op.String(), ci.OpPos)
				// avoid ; directly after tokens like ;;
				p.wroteSemi = true
			}
			p.comments(last...)
			p.flushComments()
			p.level--
		}
		p.comments(x.Last...)
		if p.swtCaseIndent {
			p.flushComments()
			p.decLevel()
		}
		p.semiRsrv("esac", x.Esac)
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
		p.spacedString("]]", x.Right)
	case *DeclClause:
		p.spacedString(x.Variant.Value, x.Pos())
		p.assigns(x.Args)
	case *TimeClause:
		p.spacedString("time", x.Pos())
		if x.PosixFormat {
			p.spacedString("-p", x.Pos())
		}
		if x.Stmt != nil {
			p.stmt(x.Stmt)
		}
	case *CoprocClause:
		p.spacedString("coproc", x.Pos())
		if x.Name != nil {
			p.space()
			p.word(x.Name)
		}
		p.space()
		p.stmt(x.Stmt)
	case *LetClause:
		p.spacedString("let", x.Pos())
		for _, n := range x.Exprs {
			p.space()
			p.arithmExpr(n, true, false)
		}
	}
	return startRedirs
}

func (p *Printer) ifClause(ic *IfClause, elif bool) {
	if !elif {
		p.spacedString("if", ic.Pos())
	}
	p.nestedStmts(ic.Cond, ic.CondLast, Pos{})
	p.semiOrNewl("then", ic.ThenPos)
	thenEnd := ic.FiPos
	el := ic.Else
	if el != nil {
		thenEnd = el.Position
	}
	p.nestedStmts(ic.Then, ic.ThenLast, thenEnd)

	if el != nil && el.ThenPos.IsValid() {
		p.comments(ic.Last...)
		p.semiRsrv("elif", el.Position)
		p.ifClause(el, true)
		return
	}
	if el == nil {
		p.comments(ic.Last...)
	} else {
		var left []Comment
		for _, c := range ic.Last {
			if c.Pos().After(el.Position) {
				left = append(left, c)
				break
			}
			p.comments(c)
		}
		p.semiRsrv("else", el.Position)
		p.comments(left...)
		p.nestedStmts(el.Then, el.ThenLast, ic.FiPos)
		p.comments(el.Last...)
	}
	p.semiRsrv("fi", ic.FiPos)
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

func (p *Printer) stmtList(stmts []*Stmt, last []Comment) {
	sep := p.wantNewline ||
		(len(stmts) > 0 && stmts[0].Pos().Line() > p.line)
	for _, s := range stmts {
		pos := s.Pos()
		var midComs, endComs []Comment
		for _, c := range s.Comments {
			if c.End().After(s.End()) {
				endComs = append(endComs, c)
				break
			}
			if c.Pos().After(pos) {
				midComs = append(midComs, c)
				continue
			}
			p.comments(c)
		}
		if !p.minify || p.wantSpace {
			p.newlines(pos)
		}
		p.line = pos.Line()
		p.comments(midComs...)
		p.stmt(s)
		p.comments(endComs...)
		p.wantNewline = true
	}
	if len(stmts) == 1 && !sep {
		p.wantNewline = false
	}
	p.comments(last...)
}

// extraIndenter ensures that all lines in a '<<-' heredoc body have at least
// baseIndent leading tabs. Those that had more tab indentation than the first
// heredoc line will keep that relative indentation.
type extraIndenter struct {
	bufWriter
	baseIndent int

	firstIndent int
	firstChange int
	curLine     []byte
}

func (e *extraIndenter) WriteByte(b byte) error {
	e.curLine = append(e.curLine, b)
	if b != '\n' {
		return nil
	}
	line := e.curLine
	if bytes.HasPrefix(e.curLine, []byte("\xff")) {
		// beginning a multiline sequence, with the leading escape
		line = line[1:]
	}
	trimmed := bytes.TrimLeft(line, "\t")
	if len(trimmed) == 1 {
		// no tabs if this is an empty line, i.e. "\n"
		e.bufWriter.Write(trimmed)
		e.curLine = e.curLine[:0]
		return nil
	}

	lineIndent := len(line) - len(trimmed)
	if e.firstIndent < 0 {
		e.firstIndent = lineIndent
		e.firstChange = e.baseIndent - lineIndent
		lineIndent = e.baseIndent
	} else {
		if lineIndent < e.firstIndent {
			lineIndent = e.firstIndent
		} else {
			lineIndent += e.firstChange
		}
	}
	for i := 0; i < lineIndent; i++ {
		e.bufWriter.WriteByte('\t')
	}
	e.bufWriter.Write(trimmed)
	e.curLine = e.curLine[:0]
	return nil
}

func (e *extraIndenter) WriteString(s string) (int, error) {
	for i := 0; i < len(s); i++ {
		e.WriteByte(s[i])
	}
	return len(s), nil
}

func (p *Printer) nestedStmts(stmts []*Stmt, last []Comment, closing Pos) {
	p.incLevel()
	switch {
	case len(stmts) > 1:
		// Force a newline if we find:
		//     { stmt; stmt; }
		p.wantNewline = true
	case closing.Line() > p.line && len(stmts) > 0 &&
		stmtsEnd(stmts, last).Line() < closing.Line():
		// Force a newline if we find:
		//     { stmt
		//     }
		p.wantNewline = true
	case len(p.pendingComments) > 0 && len(stmts) > 0:
		// Force a newline if we find:
		//     for i in a b # stmt
		//     do foo; done
		p.wantNewline = true
	}
	p.stmtList(stmts, last)
	if closing.IsValid() {
		p.flushComments()
	}
	p.decLevel()
}

func (p *Printer) assigns(assigns []*Assign) {
	p.incLevel()
	for _, a := range assigns {
		if a.Pos().Line() > p.line {
			p.bslashNewl()
		} else {
			p.spacePad(a.Pos())
		}
		if a.Name != nil {
			p.writeLit(a.Name.Value)
			p.wroteIndex(a.Index)
			if a.Append {
				p.WriteByte('+')
			}
			if !a.Naked {
				p.WriteByte('=')
			}
		}
		if a.Value != nil {
			p.word(a.Value)
		} else if a.Array != nil {
			p.wantSpace = false
			p.WriteByte('(')
			p.elemJoin(a.Array.Elems, a.Array.Last)
			p.rightParen(a.Array.Rparen)
		}
		p.wantSpace = true
	}
	p.decLevel()
}
