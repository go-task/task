// Copyright (c) 2016, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package syntax

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"sync"
	"unicode/utf8"
)

// ParseMode controls the parser behaviour via a set of flags.
type ParseMode uint

const (
	ParseComments   ParseMode = 1 << iota // add comments to the AST
	PosixConformant                       // match the POSIX standard where it differs from bash
)

var parserFree = sync.Pool{
	New: func() interface{} {
		return &parser{helperBuf: new(bytes.Buffer)}
	},
}

// Parse reads and parses a shell program with an optional name. It
// returns the parsed program if no issues were encountered. Otherwise,
// an error is returned.
func Parse(src io.Reader, name string, mode ParseMode) (*File, error) {
	p := parserFree.Get().(*parser)
	p.reset()
	alloc := &struct {
		f File
		l [32]Pos
	}{}
	p.f = &alloc.f
	p.f.Name = name
	p.f.lines = alloc.l[:1]
	p.src, p.mode = src, mode
	p.rune()
	p.next()
	p.f.Stmts = p.stmts()
	if p.err == nil {
		// EOF immediately after heredoc word so no newline to
		// trigger it
		p.doHeredocs()
	}
	f, err := p.f, p.err
	parserFree.Put(p)
	return f, err
}

type parser struct {
	src io.Reader
	bs  []byte // current chunk of read bytes
	r   rune

	f    *File
	mode ParseMode

	spaced  bool // whether tok has whitespace on its left
	newLine bool // whether tok is on a new line

	err     error // lexer/parser error
	readErr error // got a read error, but bytes left

	tok token  // current token
	val string // current value (valid if tok is _Lit*)

	pos  Pos // position of tok
	offs int // chunk offset
	npos int // pos within chunk for the next rune

	quote quoteState // current lexer state
	asPos int        // position of '=' in a literal

	forbidNested bool

	// list of pending heredoc bodies
	buriedHdocs int
	heredocs    []*Redirect
	hdocStop    []byte

	helperBuf *bytes.Buffer

	litBatch    []Lit
	wordBatch   []Word
	wpsBatch    []WordPart
	stmtBatch   []Stmt
	stListBatch []*Stmt
	callBatch   []callAlloc

	readBuf [bufSize]byte
	litBuf  [bufSize]byte
	litBs   []byte
}

const bufSize = 1 << 10

func (p *parser) reset() {
	p.bs = nil
	p.offs, p.npos = 0, 0
	p.r, p.err, p.readErr = 0, nil, nil
	p.quote, p.forbidNested = noState, false
	p.heredocs, p.buriedHdocs = p.heredocs[:0], 0
}

func (p *parser) getPos() Pos { return Pos(p.offs + p.npos) }

func (p *parser) lit(pos Pos, val string) *Lit {
	if len(p.litBatch) == 0 {
		p.litBatch = make([]Lit, 64)
	}
	l := &p.litBatch[0]
	p.litBatch = p.litBatch[1:]
	l.ValuePos = pos
	l.ValueEnd = p.getPos()
	l.Value = val
	return l
}

func (p *parser) word(parts []WordPart) *Word {
	if len(p.wordBatch) == 0 {
		p.wordBatch = make([]Word, 32)
	}
	w := &p.wordBatch[0]
	p.wordBatch = p.wordBatch[1:]
	w.Parts = parts
	return w
}

func (p *parser) wps(wp WordPart) []WordPart {
	if len(p.wpsBatch) == 0 {
		p.wpsBatch = make([]WordPart, 64)
	}
	wps := p.wpsBatch[:1:1]
	p.wpsBatch = p.wpsBatch[1:]
	wps[0] = wp
	return wps
}

func (p *parser) stmt(pos Pos) *Stmt {
	if len(p.stmtBatch) == 0 {
		p.stmtBatch = make([]Stmt, 16)
	}
	s := &p.stmtBatch[0]
	p.stmtBatch = p.stmtBatch[1:]
	s.Position = pos
	return s
}

func (p *parser) stList() []*Stmt {
	if len(p.stListBatch) == 0 {
		p.stListBatch = make([]*Stmt, 128)
	}
	stmts := p.stListBatch[:0:4]
	p.stListBatch = p.stListBatch[4:]
	return stmts
}

type callAlloc struct {
	ce CallExpr
	ws [4]*Word
}

func (p *parser) call(w *Word) *CallExpr {
	if len(p.callBatch) == 0 {
		p.callBatch = make([]callAlloc, 32)
	}
	alloc := &p.callBatch[0]
	p.callBatch = p.callBatch[1:]
	ce := &alloc.ce
	ce.Args = alloc.ws[:1]
	ce.Args[0] = w
	return ce
}

type quoteState uint

const (
	noState quoteState = 1 << iota
	subCmd
	subCmdBckquo
	sglQuotes
	dblQuotes
	hdocWord
	hdocBody
	hdocBodyTabs
	arithmExpr
	arithmExprLet
	arithmExprCmd
	arithmExprBrack
	testRegexp
	switchCase
	paramName
	paramExpName
	paramExpInd
	paramExpOff
	paramExpLen
	paramExpRepl
	paramExpExp

	allKeepSpaces = paramExpRepl | dblQuotes | hdocBody |
		hdocBodyTabs | paramExpExp | sglQuotes
	allRegTokens  = noState | subCmd | subCmdBckquo | hdocWord | switchCase
	allArithmExpr = arithmExpr | arithmExprLet | arithmExprCmd |
		arithmExprBrack | allParamArith
	allRbrack     = arithmExprBrack | paramExpInd
	allParamArith = paramExpInd | paramExpOff | paramExpLen
	allParamReg   = paramName | paramExpName | allParamArith
	allParamExp   = allParamReg | paramExpRepl | paramExpExp
)

func (p *parser) bash() bool { return p.mode&PosixConformant == 0 }

type saveState struct {
	quote       quoteState
	buriedHdocs int
}

func (p *parser) preNested(quote quoteState) (s saveState) {
	s.quote, s.buriedHdocs = p.quote, p.buriedHdocs
	p.buriedHdocs, p.quote = len(p.heredocs), quote
	return
}

func (p *parser) postNested(s saveState) {
	p.quote, p.buriedHdocs = s.quote, s.buriedHdocs
}

func (p *parser) unquotedWordBytes(w *Word) ([]byte, bool) {
	p.helperBuf.Reset()
	didUnquote := false
	for _, wp := range w.Parts {
		if p.unquotedWordPart(p.helperBuf, wp, false) {
			didUnquote = true
		}
	}
	return p.helperBuf.Bytes(), didUnquote
}

func (p *parser) unquotedWordPart(buf *bytes.Buffer, wp WordPart, quotes bool) (quoted bool) {
	switch x := wp.(type) {
	case *Lit:
		for i := 0; i < len(x.Value); i++ {
			if b := x.Value[i]; b == '\\' && !quotes {
				if i++; i < len(x.Value) {
					buf.WriteByte(x.Value[i])
				}
				quoted = true
			} else {
				buf.WriteByte(b)
			}
		}
	case *SglQuoted:
		buf.WriteString(x.Value)
		quoted = true
	case *DblQuoted:
		for _, wp2 := range x.Parts {
			p.unquotedWordPart(buf, wp2, true)
		}
		quoted = true
	}
	return
}

func (p *parser) doHeredocs() {
	old := p.quote
	hdocs := p.heredocs[p.buriedHdocs:]
	p.heredocs = p.heredocs[:p.buriedHdocs]
	for i, r := range hdocs {
		if p.err != nil {
			break
		}
		if r.Op == DashHdoc {
			p.quote = hdocBodyTabs
		} else {
			p.quote = hdocBody
		}
		var quoted bool
		p.hdocStop, quoted = p.unquotedWordBytes(r.Word)
		if i > 0 && p.r == '\n' {
			p.rune()
		}
		if quoted {
			r.Hdoc = p.hdocLitWord()
		} else {
			p.next()
			r.Hdoc = p.getWordOrEmpty()
		}
	}
	p.quote = old
}

func (p *parser) got(tok token) bool {
	if p.tok == tok {
		p.next()
		return true
	}
	return false
}

func (p *parser) gotRsrv(val string) bool {
	if p.tok == _LitWord && p.val == val {
		p.next()
		return true
	}
	return false
}

func (p *parser) gotSameLine(tok token) bool {
	if !p.newLine && p.tok == tok {
		p.next()
		return true
	}
	return false
}

func readableStr(s string) string {
	// don't quote tokens like & or }
	if s != "" && s[0] >= 'a' && s[0] <= 'z' {
		return strconv.Quote(s)
	}
	return s
}

func (p *parser) followErr(pos Pos, left, right string) {
	leftStr := readableStr(left)
	p.posErr(pos, "%s must be followed by %s", leftStr, right)
}

func (p *parser) followErrExp(pos Pos, left string) {
	p.followErr(pos, left, "an expression")
}

func (p *parser) follow(lpos Pos, left string, tok token) Pos {
	pos := p.pos
	if !p.got(tok) {
		p.followErr(lpos, left, tok.String())
	}
	return pos
}

func (p *parser) followRsrv(lpos Pos, left, val string) Pos {
	pos := p.pos
	if !p.gotRsrv(val) {
		p.followErr(lpos, left, fmt.Sprintf("%q", val))
	}
	return pos
}

func (p *parser) followStmts(left string, lpos Pos, stops ...string) []*Stmt {
	if p.gotSameLine(semicolon) {
		return nil
	}
	sts := p.stmts(stops...)
	if len(sts) < 1 && !p.newLine {
		p.followErr(lpos, left, "a statement list")
	}
	return sts
}

func (p *parser) followWordTok(tok token, pos Pos) *Word {
	w := p.getWord()
	if w == nil {
		p.followErr(pos, tok.String(), "a word")
	}
	return w
}

func (p *parser) followWord(s string, pos Pos) *Word {
	w := p.getWord()
	if w == nil {
		p.followErr(pos, s, "a word")
	}
	return w
}

func (p *parser) stmtEnd(n Node, start, end string) Pos {
	pos := p.pos
	if !p.gotRsrv(end) {
		p.posErr(n.Pos(), "%s statement must end with %q", start, end)
	}
	return pos
}

func (p *parser) quoteErr(lpos Pos, quote token) {
	p.posErr(lpos, "reached %s without closing quote %s",
		p.tok.String(), quote)
}

func (p *parser) matchingErr(lpos Pos, left, right interface{}) {
	p.posErr(lpos, "reached %s without matching %s with %s",
		p.tok.String(), left, right)
}

func (p *parser) matched(lpos Pos, left, right token) Pos {
	pos := p.pos
	if !p.got(right) {
		p.matchingErr(lpos, left, right)
	}
	return pos
}

func (p *parser) errPass(err error) {
	if p.err == nil {
		p.err = err
		p.npos = len(p.bs) + 1
		p.r = utf8.RuneSelf
		p.tok = _EOF
	}
}

// ParseError represents an error found when parsing a source file.
type ParseError struct {
	Position
	Text string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%s: %s", e.Position.String(), e.Text)
}

func (p *parser) posErr(pos Pos, format string, a ...interface{}) {
	p.errPass(&ParseError{
		Position: p.f.Position(pos),
		Text:     fmt.Sprintf(format, a...),
	})
}

func (p *parser) curErr(format string, a ...interface{}) {
	p.posErr(p.pos, format, a...)
}

func (p *parser) stmts(stops ...string) (sts []*Stmt) {
	gotEnd := true
	for p.tok != _EOF {
		switch p.tok {
		case _LitWord:
			for _, stop := range stops {
				if p.val == stop {
					return
				}
			}
		case rightParen:
			if p.quote == subCmd {
				return
			}
		case bckQuote:
			if p.quote == subCmdBckquo {
				return
			}
		case dblSemicolon, semiFall, dblSemiFall:
			if p.quote == switchCase {
				return
			}
			p.curErr("%s can only be used in a case clause", p.tok)
		}
		if !p.newLine && !gotEnd {
			p.curErr("statements must be separated by &, ; or a newline")
		}
		if p.tok == _EOF {
			break
		}
		if s, end := p.getStmt(true, false); s == nil {
			p.invalidStmtStart()
		} else {
			if sts == nil {
				sts = p.stList()
			}
			sts = append(sts, s)
			gotEnd = end
		}
	}
	return
}

func (p *parser) invalidStmtStart() {
	switch p.tok {
	case semicolon, and, or, andAnd, orOr:
		p.curErr("%s can only immediately follow a statement", p.tok)
	case rightParen:
		p.curErr("%s can only be used to close a subshell", p.tok)
	default:
		p.curErr("%s is not a valid start for a statement", p.tok)
	}
}

func (p *parser) getWord() *Word {
	if parts := p.wordParts(); len(parts) > 0 {
		return p.word(parts)
	}
	return nil
}

func (p *parser) getWordOrEmpty() *Word {
	parts := p.wordParts()
	if len(parts) == 0 {
		l := p.lit(p.pos, "")
		l.ValueEnd = l.ValuePos // force Lit.Pos() == Lit.End()
		return p.word(p.wps(l))
	}
	return p.word(parts)
}

func (p *parser) getLit() *Lit {
	switch p.tok {
	case _Lit, _LitWord, _LitRedir:
		l := p.lit(p.pos, p.val)
		p.next()
		return l
	}
	return nil
}

func (p *parser) wordParts() (wps []WordPart) {
	for {
		n := p.wordPart()
		if n == nil {
			return
		}
		if wps == nil {
			wps = p.wps(n)
		} else {
			wps = append(wps, n)
		}
		if p.spaced {
			return
		}
	}
}

func (p *parser) ensureNoNested() {
	if p.forbidNested {
		p.curErr("expansions not allowed in heredoc words")
	}
}

func (p *parser) wordPart() WordPart {
	switch p.tok {
	case _Lit, _LitWord:
		l := p.lit(p.pos, p.val)
		p.next()
		return l
	case dollBrace:
		p.ensureNoNested()
		return p.paramExp()
	case dollDblParen, dollBrack:
		p.ensureNoNested()
		left := p.tok
		ar := &ArithmExp{Left: p.pos, Bracket: left == dollBrack}
		var old saveState
		if ar.Bracket {
			old = p.preNested(arithmExprBrack)
		} else {
			old = p.preNested(arithmExpr)
		}
		p.next()
		ar.X = p.arithmExpr(left, ar.Left, 0, false, false)
		if ar.Bracket {
			if p.tok != rightBrack {
				p.matchingErr(ar.Left, dollBrack, rightBrack)
			}
			p.postNested(old)
			ar.Right = p.pos
			p.next()
		} else {
			ar.Right = p.arithmEnd(dollDblParen, ar.Left, old)
		}
		return ar
	case dollParen:
		p.ensureNoNested()
		cs := &CmdSubst{Left: p.pos}
		old := p.preNested(subCmd)
		p.next()
		cs.Stmts = p.stmts()
		p.postNested(old)
		cs.Right = p.matched(cs.Left, leftParen, rightParen)
		return cs
	case dollar:
		r := p.r
		if r == utf8.RuneSelf || wordBreak(r) || r == '"' || r == '\'' || r == '`' || r == '[' {
			l := p.lit(p.pos, "$")
			p.next()
			return l
		}
		p.ensureNoNested()
		pe := &ParamExp{Dollar: p.pos, Short: true}
		p.pos++
		switch r {
		case '@', '*', '#', '$', '?', '!', '0', '-':
			p.rune()
			p.tok, p.val = _LitWord, string(r)
		default:
			old := p.quote
			p.quote = paramName
			p.advanceLitOther(r)
			p.quote = old
		}
		pe.Param = p.getLit()
		return pe
	case cmdIn, cmdOut:
		p.ensureNoNested()
		ps := &ProcSubst{Op: ProcOperator(p.tok), OpPos: p.pos}
		old := p.preNested(subCmd)
		p.next()
		ps.Stmts = p.stmts()
		p.postNested(old)
		ps.Rparen = p.matched(ps.OpPos, token(ps.Op), rightParen)
		return ps
	case sglQuote:
		sq := &SglQuoted{Position: p.pos}
		r := p.r
	loop:
		for p.newLit(r); ; r = p.rune() {
			switch r {
			case utf8.RuneSelf, '\'':
				sq.Value = p.endLit()
				p.rune()
				break loop
			}
		}
		if r != '\'' {
			p.posErr(sq.Pos(), "reached EOF without closing quote %s", sglQuote)
		}
		p.next()
		return sq
	case dollSglQuote:
		sq := &SglQuoted{Position: p.pos, Dollar: true}
		old := p.quote
		p.quote = sglQuotes
		p.next()
		p.quote = old
		if p.tok != sglQuote {
			sq.Value = p.val
			p.next()
		}
		if !p.got(sglQuote) {
			p.quoteErr(sq.Pos(), sglQuote)
		}
		return sq
	case dblQuote:
		if p.quote == dblQuotes {
			return nil
		}
		fallthrough
	case dollDblQuote:
		q := &DblQuoted{Position: p.pos, Dollar: p.tok == dollDblQuote}
		old := p.quote
		p.quote = dblQuotes
		p.next()
		q.Parts = p.wordParts()
		p.quote = old
		if !p.got(dblQuote) {
			p.quoteErr(q.Pos(), dblQuote)
		}
		return q
	case bckQuote:
		if p.quote == subCmdBckquo {
			return nil
		}
		p.ensureNoNested()
		cs := &CmdSubst{Left: p.pos}
		old := p.preNested(subCmdBckquo)
		p.next()
		cs.Stmts = p.stmts()
		p.postNested(old)
		cs.Right = p.pos
		if !p.got(bckQuote) {
			p.quoteErr(cs.Pos(), bckQuote)
		}
		return cs
	case globQuest, globStar, globPlus, globAt, globExcl:
		if !p.bash() {
			p.curErr("extended globs are a bash feature")
		}
		eg := &ExtGlob{Op: GlobOperator(p.tok), OpPos: p.pos}
		lparens := 0
		r := p.r
	globLoop:
		for p.newLit(r); ; r = p.rune() {
			switch r {
			case utf8.RuneSelf:
				break globLoop
			case '(':
				lparens++
			case ')':
				if lparens--; lparens < 0 {
					break globLoop
				}
			}
		}
		eg.Pattern = p.lit(eg.OpPos+2, p.endLit())
		p.rune()
		p.next()
		if lparens != -1 {
			p.matchingErr(eg.OpPos, eg.Op, rightParen)
		}
		return eg
	default:
		return nil
	}
}

func arithmOpLevel(op BinAritOperator) int {
	switch op {
	case Comma:
		return 0
	case AddAssgn, SubAssgn, MulAssgn, QuoAssgn, RemAssgn, AndAssgn,
		OrAssgn, XorAssgn, ShlAssgn, ShrAssgn:
		return 1
	case Assgn:
		return 2
	case Quest, Colon:
		return 3
	case AndArit, OrArit:
		return 4
	case And, Or, Xor:
		return 5
	case Eql, Neq:
		return 6
	case Lss, Gtr, Leq, Geq:
		return 7
	case Shl, Shr:
		return 8
	case Add, Sub:
		return 9
	case Mul, Quo, Rem:
		return 10
	case Pow:
		return 11
	}
	return -1
}

func (p *parser) arithmExpr(ftok token, fpos Pos, level int, compact, tern bool) ArithmExpr {
	if p.tok == _EOF || p.peekArithmEnd() {
		return nil
	}
	var left ArithmExpr
	if level > 11 {
		left = p.arithmExprBase(compact)
	} else {
		left = p.arithmExpr(ftok, fpos, level+1, compact, false)
	}
	if compact && p.spaced {
		return left
	}
	newLevel := arithmOpLevel(BinAritOperator(p.tok))
	if !tern && p.tok == colon && p.quote&allParamArith != 0 {
		newLevel = -1
	}
	if newLevel < 0 {
		switch p.tok {
		case _Lit, _LitWord:
			p.curErr("not a valid arithmetic operator: %s", p.val)
			return nil
		case rightParen, _EOF:
		default:
			if p.quote == arithmExpr {
				p.curErr("not a valid arithmetic operator: %v", p.tok)
				return nil
			}
		}
	}
	if newLevel < level {
		return left
	}
	if left == nil {
		p.curErr("%s must follow an expression", p.tok.String())
		return nil
	}
	b := &BinaryArithm{
		OpPos: p.pos,
		Op:    BinAritOperator(p.tok),
		X:     left,
	}
	switch b.Op {
	case Colon:
		if !tern {
			p.posErr(b.Pos(), "ternary operator missing ? before :")
		}
	case AddAssgn, SubAssgn, MulAssgn, QuoAssgn, RemAssgn, AndAssgn,
		OrAssgn, XorAssgn, ShlAssgn, ShrAssgn, Assgn:
		if l, ok := b.X.(*Lit); !ok || !validIdent(l.Value, p.bash()) {
			p.posErr(b.OpPos, "%s must follow a name", b.Op.String())
		}
	}
	if p.next(); compact && p.spaced {
		p.followErrExp(b.OpPos, b.Op.String())
	}
	b.Y = p.arithmExpr(token(b.Op), b.OpPos, newLevel, compact, b.Op == Quest)
	if b.Y == nil {
		p.followErrExp(b.OpPos, b.Op.String())
	}
	if b.Op == Quest {
		if b2, ok := b.Y.(*BinaryArithm); !ok || b2.Op != Colon {
			p.posErr(b.Pos(), "ternary operator missing : after ?")
		}
	}
	return b
}

func (p *parser) arithmExprBase(compact bool) ArithmExpr {
	var x ArithmExpr
	switch p.tok {
	case exclMark:
		ue := &UnaryArithm{OpPos: p.pos, Op: UnAritOperator(p.tok)}
		p.next()
		if ue.X = p.arithmExprBase(compact); ue.X == nil {
			p.followErrExp(ue.OpPos, ue.Op.String())
		}
		return ue
	case addAdd, subSub:
		ue := &UnaryArithm{OpPos: p.pos, Op: UnAritOperator(p.tok)}
		p.next()
		if lit := p.getLit(); lit == nil {
			p.followErr(ue.OpPos, token(ue.Op).String(), "a literal")
		} else {
			ue.X = lit
		}
		return ue
	case leftParen:
		pe := &ParenArithm{Lparen: p.pos}
		p.next()
		pe.X = p.arithmExpr(leftParen, pe.Lparen, 0, false, false)
		if pe.X == nil {
			p.posErr(pe.Lparen, "parentheses must enclose an expression")
		}
		pe.Rparen = p.matched(pe.Lparen, leftParen, rightParen)
		x = pe
	case plus, minus:
		ue := &UnaryArithm{OpPos: p.pos, Op: UnAritOperator(p.tok)}
		if p.next(); compact && p.spaced {
			p.followErrExp(ue.OpPos, ue.Op.String())
		}
		ue.X = p.arithmExpr(token(ue.Op), ue.OpPos, 0, compact, false)
		if ue.X == nil {
			p.followErrExp(ue.OpPos, ue.Op.String())
		}
		x = ue
	case illegalTok, rightBrack, rightBrace, rightParen:
	case _LitWord:
		x = p.getLit()
	case dollar, dollBrace:
		x = p.wordPart().(*ParamExp)
	case bckQuote:
		if p.quote == arithmExprLet {
			return nil
		}
		fallthrough
	default:
		if arithmOpLevel(BinAritOperator(p.tok)) >= 0 {
			break
		}
		p.curErr("arithmetic expressions must consist of names and numbers")
	}
	if compact && p.spaced {
		return x
	}
	if p.tok == addAdd || p.tok == subSub {
		if l, ok := x.(*Lit); !ok || !validIdent(l.Value, p.bash()) {
			p.curErr("%s must follow a name", p.tok.String())
		}
		u := &UnaryArithm{
			Post:  true,
			OpPos: p.pos,
			Op:    UnAritOperator(p.tok),
			X:     x,
		}
		p.next()
		return u
	}
	return x
}

func (p *parser) paramExp() *ParamExp {
	pe := &ParamExp{Dollar: p.pos}
	old := p.quote
	p.quote = paramExpName
	p.next()
	switch p.tok {
	case at:
		p.tok, p.val = _LitWord, "@"
	case dblHash:
		p.tok = hash
		p.unrune('#')
		fallthrough
	case hash:
		if p.r != '}' {
			pe.Length = true
			p.next()
		}
	case exclMark:
		if p.r != '}' {
			pe.Indirect = true
			p.next()
		}
	}
	switch p.tok {
	case _Lit, _LitWord:
		pe.Param = p.lit(p.pos, p.val)
		p.next()
	case hash, exclMark:
		pe.Param = p.lit(p.pos, p.tok.String())
		p.next()
	case dollar, quest, minus:
		op := p.tok
		pe.Param = p.lit(p.pos, p.tok.String())
		p.next()
		switch p.tok {
		case _Lit, _LitWord:
			p.curErr("%s cannot be followed by a word", op)
		}
	default:
		if !pe.Length {
			p.posErr(pe.Dollar, "parameter expansion requires a literal")
		}
	}
	if p.tok == rightBrace {
		pe.Rbrace = p.pos
		p.quote = old
		p.next()
		return pe
	}
	if p.tok == leftBrack {
		if !p.bash() {
			p.curErr("arrays are a bash feature")
		}
		lpos := p.pos
		p.quote = paramExpInd
		p.next()
		switch p.tok {
		case star:
			p.tok, p.val = _LitWord, "*"
		case at:
			p.tok, p.val = _LitWord, "@"
		}
		pe.Ind = &Index{
			Expr: p.arithmExpr(leftBrack, lpos, 0, false, false),
		}
		if pe.Ind.Expr == nil {
			p.followErrExp(lpos, "[")
		}
		p.quote = paramExpName
		p.matched(lpos, leftBrack, rightBrack)
	}
	switch p.tok {
	case rightBrace:
		pe.Rbrace = p.pos
		p.quote = old
		p.next()
		return pe
	case slash, dblSlash:
		if !p.bash() {
			p.curErr("search and replace is a bash feature")
		}
		pe.Repl = &Replace{All: p.tok == dblSlash}
		p.quote = paramExpRepl
		p.next()
		pe.Repl.Orig = p.getWordOrEmpty()
		switch p.tok {
		case dblSlash:
			p.unrune('/')
			fallthrough
		case slash:
			p.quote = paramExpExp
			p.next()
		}
		pe.Repl.With = p.getWordOrEmpty()
	case colon:
		if !p.bash() {
			p.curErr("slicing is a bash feature")
		}
		pe.Slice = &Slice{}
		colonPos := p.pos
		p.quote = paramExpOff
		if p.next(); p.tok != colon {
			pe.Slice.Offset = p.arithmExpr(colon, colonPos, 0, false, false)
			if pe.Slice.Offset == nil {
				p.followErrExp(colonPos, ":")
			}
		}
		colonPos = p.pos
		p.quote = paramExpLen
		if p.got(colon) {
			pe.Slice.Length = p.arithmExpr(colon, colonPos, 0, false, false)
			if pe.Slice.Length == nil {
				p.followErrExp(colonPos, ":")
			}
		}
	case caret, dblCaret, comma, dblComma, at:
		if !p.bash() {
			p.curErr("this expansion operator is a bash feature")
		}
		fallthrough
	default:
		pe.Exp = &Expansion{Op: ParExpOperator(p.tok)}
		p.quote = paramExpExp
		p.next()
		pe.Exp.Word = p.getWordOrEmpty()
	}
	p.quote = old
	pe.Rbrace = p.pos
	p.matched(pe.Dollar, dollBrace, rightBrace)
	return pe
}

func (p *parser) peekArithmEnd() bool {
	return p.tok == rightParen && p.r == ')'
}

func (p *parser) arithmEnd(ltok token, lpos Pos, old saveState) Pos {
	if p.peekArithmEnd() {
		p.rune()
	} else {
		p.matchingErr(lpos, ltok, dblRightParen)
	}
	p.postNested(old)
	pos := p.pos
	p.next()
	return pos
}

func stopToken(tok token) bool {
	switch tok {
	case _EOF, semicolon, and, or, andAnd, orOr, pipeAll, dblSemicolon,
		semiFall, dblSemiFall, rightParen:
		return true
	}
	return false
}

func validIdent(val string, bash bool) bool {
	for i, c := range val {
		switch {
		case 'a' <= c && c <= 'z':
		case 'A' <= c && c <= 'Z':
		case c == '_':
		case i > 0 && '0' <= c && c <= '9':
		case i > 0 && (c == '[' || c == ']') && bash:
		case c == '+' && i == len(val)-1 && bash:
		default:
			return false
		}
	}
	return true
}

func (p *parser) hasValidIdent() bool {
	if p.asPos < 1 {
		return false
	}
	return validIdent(p.val[:p.asPos], p.bash())
}

func (p *parser) getAssign() *Assign {
	as := &Assign{}
	nameEnd := p.asPos
	if p.bash() && p.val[p.asPos-1] == '+' {
		// a+=b
		as.Append = true
		nameEnd--
	}
	as.Name = p.lit(p.pos, p.val[:nameEnd])
	// since we're not using the entire p.val
	as.Name.ValueEnd = as.Name.ValuePos + Pos(nameEnd)
	start := p.lit(p.pos+1, p.val[p.asPos+1:])
	if start.Value != "" {
		start.ValuePos += Pos(p.asPos)
		as.Value = p.word(p.wps(start))
	}
	if p.next(); p.spaced {
		return as
	}
	if start.Value == "" && p.tok == leftParen {
		if !p.bash() {
			p.curErr("arrays are a bash feature")
		}
		ae := &ArrayExpr{Lparen: p.pos}
		p.next()
		for p.tok != _EOF && p.tok != rightParen {
			if w := p.getWord(); w == nil {
				p.curErr("array elements must be words")
			} else {
				ae.List = append(ae.List, w)
			}
		}
		ae.Rparen = p.matched(ae.Lparen, leftParen, rightParen)
		as.Value = p.word(p.wps(ae))
	} else if !p.newLine && !stopToken(p.tok) {
		if w := p.getWord(); w != nil {
			if as.Value == nil {
				as.Value = w
			} else {
				as.Value.Parts = append(as.Value.Parts, w.Parts...)
			}
		}
	}
	return as
}

func (p *parser) peekRedir() bool {
	switch p.tok {
	case rdrOut, appOut, rdrIn, dplIn, dplOut, clbOut, rdrInOut,
		hdoc, dashHdoc, wordHdoc, rdrAll, appAll, _LitRedir:
		return true
	}
	return false
}

func (p *parser) doRedirect(s *Stmt) {
	r := &Redirect{}
	r.N = p.getLit()
	r.Op, r.OpPos = RedirOperator(p.tok), p.pos
	p.next()
	if p.newLine {
		p.curErr("redirect word must be on the same line")
	}
	switch r.Op {
	case Hdoc, DashHdoc:
		old := p.quote
		p.quote, p.forbidNested = hdocWord, true
		p.heredocs = append(p.heredocs, r)
		r.Word = p.followWordTok(token(r.Op), r.OpPos)
		p.quote, p.forbidNested = old, false
		if p.tok == illegalTok {
			p.next()
		}
	default:
		r.Word = p.followWordTok(token(r.Op), r.OpPos)
	}
	s.Redirs = append(s.Redirs, r)
}

func (p *parser) getStmt(readEnd, binCmd bool) (s *Stmt, gotEnd bool) {
	s = p.stmt(p.pos)
	if p.gotRsrv("!") {
		s.Negated = true
	}
preLoop:
	for {
		switch p.tok {
		case _Lit, _LitWord:
			if p.hasValidIdent() {
				s.Assigns = append(s.Assigns, p.getAssign())
			} else {
				break preLoop
			}
		case rdrOut, appOut, rdrIn, dplIn, dplOut, clbOut, rdrInOut,
			hdoc, dashHdoc, wordHdoc, rdrAll, appAll, _LitRedir:
			p.doRedirect(s)
		default:
			break preLoop
		}
		switch {
		case p.newLine, p.tok == _EOF:
			return
		case p.tok == semicolon:
			if readEnd {
				s.Semicolon = p.pos
				p.next()
				gotEnd = true
			}
			return
		}
	}
	if s = p.gotStmtPipe(s); s == nil {
		return
	}
	switch p.tok {
	case andAnd, orOr:
		// left associativity: in a list of BinaryCmds, the
		// right recursion should only read a single element.
		if binCmd {
			return
		}
		// and instead of using recursion, iterate manually
		for p.tok == andAnd || p.tok == orOr {
			b := &BinaryCmd{
				OpPos: p.pos,
				Op:    BinCmdOperator(p.tok),
				X:     s,
			}
			p.next()
			if b.Y, _ = p.getStmt(false, true); b.Y == nil {
				p.followErr(b.OpPos, b.Op.String(), "a statement")
			}
			s = p.stmt(s.Position)
			s.Cmd = b
		}
		if readEnd && p.gotSameLine(semicolon) {
			gotEnd = true
		}
	case and:
		p.next()
		s.Background = true
		gotEnd = true
	case semicolon:
		if !p.newLine && readEnd {
			s.Semicolon = p.pos
			p.next()
			gotEnd = true
		}
	}
	return
}

func (p *parser) gotStmtPipe(s *Stmt) *Stmt {
	switch p.tok {
	case _LitWord:
		switch p.val {
		case "{":
			s.Cmd = p.block()
		case "if":
			s.Cmd = p.ifClause()
		case "while":
			s.Cmd = p.whileClause()
		case "until":
			s.Cmd = p.untilClause()
		case "for":
			s.Cmd = p.forClause()
		case "case":
			s.Cmd = p.caseClause()
		case "}":
			p.curErr(`%s can only be used to close a block`, p.val)
		case "]]":
			if !p.bash() {
				break
			}
			p.curErr(`%s can only be used to close a test`, p.val)
		case "then":
			p.curErr(`%q can only be used in an if`, p.val)
		case "elif":
			p.curErr(`%q can only be used in an if`, p.val)
		case "fi":
			p.curErr(`%q can only be used to end an if`, p.val)
		case "do":
			p.curErr(`%q can only be used in a loop`, p.val)
		case "done":
			p.curErr(`%q can only be used to end a loop`, p.val)
		case "esac":
			p.curErr(`%q can only be used to end a case`, p.val)
		default:
			if !p.bash() {
				break
			}
			switch p.val {
			case "[[":
				s.Cmd = p.testClause()
			case "declare", "local", "export", "readonly",
				"typeset", "nameref":
				s.Cmd = p.declClause()
			case "coproc":
				s.Cmd = p.coprocClause()
			case "let":
				s.Cmd = p.letClause()
			case "function":
				s.Cmd = p.bashFuncDecl()
			}
		}
		if s.Cmd == nil {
			name := p.lit(p.pos, p.val)
			if p.next(); p.gotSameLine(leftParen) {
				p.follow(name.ValuePos, "foo(", rightParen)
				s.Cmd = p.funcDecl(name, name.ValuePos)
			} else {
				s.Cmd = p.callExpr(s, p.word(p.wps(name)))
			}
		}
	case bckQuote:
		if p.quote == subCmdBckquo {
			return s
		}
		fallthrough
	case _Lit, dollBrace, dollDblParen, dollParen, dollar, cmdIn, cmdOut,
		sglQuote, dollSglQuote, dblQuote, dollDblQuote, dollBrack,
		globQuest, globStar, globPlus, globAt, globExcl:
		w := p.word(p.wordParts())
		if p.gotSameLine(leftParen) && p.err == nil {
			p.posErr(w.Pos(), "invalid func name")
		}
		s.Cmd = p.callExpr(s, w)
	case leftParen:
		s.Cmd = p.subshell()
	case dblLeftParen:
		s.Cmd = p.arithmExpCmd()
	}
	for !p.newLine && p.peekRedir() {
		p.doRedirect(s)
	}
	if s.Cmd == nil && len(s.Redirs) == 0 && !s.Negated && len(s.Assigns) == 0 {
		return nil
	}
	if p.tok == or || p.tok == pipeAll {
		b := &BinaryCmd{OpPos: p.pos, Op: BinCmdOperator(p.tok), X: s}
		p.next()
		if b.Y = p.gotStmtPipe(p.stmt(p.pos)); b.Y == nil {
			p.followErr(b.OpPos, b.Op.String(), "a statement")
		}
		s = p.stmt(s.Position)
		s.Cmd = b
	}
	return s
}

func (p *parser) subshell() *Subshell {
	s := &Subshell{Lparen: p.pos}
	old := p.preNested(subCmd)
	p.next()
	s.Stmts = p.stmts()
	p.postNested(old)
	s.Rparen = p.matched(s.Lparen, leftParen, rightParen)
	return s
}

func (p *parser) arithmExpCmd() Command {
	ar := &ArithmCmd{Left: p.pos}
	old := p.preNested(arithmExprCmd)
	p.next()
	ar.X = p.arithmExpr(dblLeftParen, ar.Left, 0, false, false)
	ar.Right = p.arithmEnd(dblLeftParen, ar.Left, old)
	return ar
}

func (p *parser) block() *Block {
	b := &Block{Lbrace: p.pos}
	p.next()
	b.Stmts = p.stmts("}")
	b.Rbrace = p.pos
	if !p.gotRsrv("}") {
		p.matchingErr(b.Lbrace, "{", "}")
	}
	return b
}

func (p *parser) ifClause() *IfClause {
	ic := &IfClause{If: p.pos}
	p.next()
	ic.CondStmts = p.followStmts("if", ic.If, "then")
	ic.Then = p.followRsrv(ic.If, "if <cond>", "then")
	ic.ThenStmts = p.followStmts("then", ic.Then, "fi", "elif", "else")
	for p.tok == _LitWord && p.val == "elif" {
		elf := &Elif{Elif: p.pos}
		p.next()
		elf.CondStmts = p.followStmts("elif", elf.Elif, "then")
		elf.Then = p.followRsrv(elf.Elif, "elif <cond>", "then")
		elf.ThenStmts = p.followStmts("then", elf.Then, "fi", "elif", "else")
		ic.Elifs = append(ic.Elifs, elf)
	}
	if elsePos := p.pos; p.gotRsrv("else") {
		ic.Else = elsePos
		ic.ElseStmts = p.followStmts("else", ic.Else, "fi")
	}
	ic.Fi = p.stmtEnd(ic, "if", "fi")
	return ic
}

func (p *parser) whileClause() *WhileClause {
	wc := &WhileClause{While: p.pos}
	p.next()
	wc.CondStmts = p.followStmts("while", wc.While, "do")
	wc.Do = p.followRsrv(wc.While, "while <cond>", "do")
	wc.DoStmts = p.followStmts("do", wc.Do, "done")
	wc.Done = p.stmtEnd(wc, "while", "done")
	return wc
}

func (p *parser) untilClause() *UntilClause {
	uc := &UntilClause{Until: p.pos}
	p.next()
	uc.CondStmts = p.followStmts("until", uc.Until, "do")
	uc.Do = p.followRsrv(uc.Until, "until <cond>", "do")
	uc.DoStmts = p.followStmts("do", uc.Do, "done")
	uc.Done = p.stmtEnd(uc, "until", "done")
	return uc
}

func (p *parser) forClause() *ForClause {
	fc := &ForClause{For: p.pos}
	p.next()
	fc.Loop = p.loop(fc.For)
	fc.Do = p.followRsrv(fc.For, "for foo [in words]", "do")
	fc.DoStmts = p.followStmts("do", fc.Do, "done")
	fc.Done = p.stmtEnd(fc, "for", "done")
	return fc
}

func (p *parser) loop(forPos Pos) Loop {
	if p.tok == dblLeftParen {
		cl := &CStyleLoop{Lparen: p.pos}
		old := p.preNested(arithmExprCmd)
		if p.next(); p.tok == dblSemicolon {
			p.unrune(';')
			p.tok = semicolon
		}
		if p.tok != semicolon {
			cl.Init = p.arithmExpr(dblLeftParen, cl.Lparen, 0, false, false)
		}
		scPos := p.pos
		if p.tok == dblSemicolon {
			p.unrune(';')
			p.tok = semicolon
		}
		p.follow(p.pos, "expression", semicolon)
		if p.tok != semicolon {
			cl.Cond = p.arithmExpr(semicolon, scPos, 0, false, false)
		}
		scPos = p.pos
		p.follow(p.pos, "expression", semicolon)
		if p.tok != semicolon {
			cl.Post = p.arithmExpr(semicolon, scPos, 0, false, false)
		}
		cl.Rparen = p.arithmEnd(dblLeftParen, cl.Lparen, old)
		p.gotSameLine(semicolon)
		return cl
	}
	wi := &WordIter{}
	if wi.Name = p.getLit(); wi.Name == nil {
		p.followErr(forPos, "for", "a literal")
	}
	if p.gotRsrv("in") {
		for !p.newLine && p.tok != _EOF && p.tok != semicolon {
			if w := p.getWord(); w == nil {
				p.curErr("word list can only contain words")
			} else {
				wi.List = append(wi.List, w)
			}
		}
		p.gotSameLine(semicolon)
	} else if !p.newLine && !p.got(semicolon) {
		p.followErr(forPos, "for foo", `"in", ; or a newline`)
	}
	return wi
}

func (p *parser) caseClause() *CaseClause {
	cc := &CaseClause{Case: p.pos}
	p.next()
	cc.Word = p.followWord("case", cc.Case)
	p.followRsrv(cc.Case, "case x", "in")
	cc.List = p.patLists()
	cc.Esac = p.stmtEnd(cc, "case", "esac")
	return cc
}

func (p *parser) patLists() (pls []*PatternList) {
	for p.tok != _EOF && !(p.tok == _LitWord && p.val == "esac") {
		pl := &PatternList{}
		p.got(leftParen)
		for p.tok != _EOF {
			if w := p.getWord(); w == nil {
				p.curErr("case patterns must consist of words")
			} else {
				pl.Patterns = append(pl.Patterns, w)
			}
			if p.tok == rightParen {
				break
			}
			if !p.got(or) {
				p.curErr("case patterns must be separated with |")
			}
		}
		old := p.preNested(switchCase)
		p.next()
		pl.Stmts = p.stmts("esac")
		p.postNested(old)
		pl.OpPos = p.pos
		if p.tok != dblSemicolon && p.tok != semiFall && p.tok != dblSemiFall {
			pl.Op = DblSemicolon
			pls = append(pls, pl)
			break
		}
		pl.Op = CaseOperator(p.tok)
		p.next()
		pls = append(pls, pl)
	}
	return
}

func (p *parser) testClause() *TestClause {
	tc := &TestClause{Left: p.pos}
	if p.next(); p.tok == _EOF || p.gotRsrv("]]") {
		p.posErr(tc.Left, "test clause requires at least one expression")
	}
	tc.X = p.testExpr(illegalTok, tc.Left, 0)
	tc.Right = p.pos
	if !p.gotRsrv("]]") {
		p.matchingErr(tc.Left, "[[", "]]")
	}
	return tc
}

func (p *parser) testExpr(ftok token, fpos Pos, level int) TestExpr {
	var left TestExpr
	if level > 1 {
		left = p.testExprBase(ftok, fpos)
	} else {
		left = p.testExpr(ftok, fpos, level+1)
	}
	if left == nil {
		return left
	}
	var newLevel int
	switch p.tok {
	case andAnd, orOr:
	case _LitWord:
		if p.val == "]]" {
			return left
		}
		fallthrough
	case rdrIn, rdrOut:
		newLevel = 1
	case _EOF, rightParen:
		return left
	case _Lit:
		p.curErr("not a valid test operator: %s", p.val)
	default:
		p.curErr("not a valid test operator: %v", p.tok)
	}
	if newLevel < level {
		return left
	}
	if p.tok == _LitWord {
		if p.tok = testBinaryOp(p.val); p.tok == illegalTok {
			p.curErr("not a valid test operator: %s", p.val)
		}
	}
	b := &BinaryTest{
		OpPos: p.pos,
		Op:    BinTestOperator(p.tok),
		X:     left,
	}
	switch b.Op {
	case AndTest, OrTest:
		p.next()
		if b.Y = p.testExpr(token(b.Op), b.OpPos, newLevel); b.Y == nil {
			p.followErrExp(b.OpPos, b.Op.String())
		}
	case TsReMatch:
		old := p.preNested(testRegexp)
		defer p.postNested(old)
		fallthrough
	default:
		if _, ok := b.X.(*Word); !ok {
			p.posErr(b.OpPos, "expected %s, %s or %s after complex expr",
				AndTest, OrTest, "]]")
		}
		p.next()
		b.Y = p.followWordTok(token(b.Op), b.OpPos)
	}
	return b
}

func (p *parser) testExprBase(ftok token, fpos Pos) TestExpr {
	switch p.tok {
	case _EOF:
		return nil
	case _LitWord:
		if op := testUnaryOp(p.val); op != illegalTok {
			p.tok = op
		}
	}
	switch p.tok {
	case exclMark:
		u := &UnaryTest{OpPos: p.pos, Op: TsNot}
		p.next()
		u.X = p.testExpr(token(u.Op), u.OpPos, 0)
		return u
	case tsExists, tsRegFile, tsDirect, tsCharSp, tsBlckSp, tsNmPipe,
		tsSocket, tsSmbLink, tsSticky, tsGIDSet, tsUIDSet, tsGrpOwn,
		tsUsrOwn, tsModif, tsRead, tsWrite, tsExec, tsNoEmpty,
		tsFdTerm, tsEmpStr, tsNempStr, tsOptSet, tsVarSet, tsRefVar:
		u := &UnaryTest{OpPos: p.pos, Op: UnTestOperator(p.tok)}
		p.next()
		u.X = p.followWordTok(ftok, fpos)
		return u
	case leftParen:
		pe := &ParenTest{Lparen: p.pos}
		p.next()
		if pe.X = p.testExpr(leftParen, pe.Lparen, 0); pe.X == nil {
			p.posErr(pe.Lparen, "parentheses must enclose an expression")
		}
		pe.Rparen = p.matched(pe.Lparen, leftParen, rightParen)
		return pe
	case rightParen:
		return nil
	default:
		// since we don't have [[ as a token
		fstr := "[["
		if ftok != illegalTok {
			fstr = ftok.String()
		}
		return p.followWord(fstr, fpos)
	}
}

func (p *parser) declClause() *DeclClause {
	name := p.val
	ds := &DeclClause{Position: p.pos}
	switch name {
	case "declare", "typeset": // typeset is an obsolete synonym
	default:
		ds.Variant = name
	}
	p.next()
	for p.tok == _LitWord && p.val[0] == '-' {
		ds.Opts = append(ds.Opts, p.getWord())
	}
	for !p.newLine && !stopToken(p.tok) && !p.peekRedir() {
		if (p.tok == _Lit || p.tok == _LitWord) && p.hasValidIdent() {
			ds.Assigns = append(ds.Assigns, p.getAssign())
		} else if w := p.getWord(); w == nil {
			p.followErr(p.pos, name, "words")
		} else {
			ds.Assigns = append(ds.Assigns, &Assign{Value: w})
		}
	}
	return ds
}

func isBashCompoundCommand(tok token, val string) bool {
	switch tok {
	case leftParen, dblLeftParen:
		return true
	case _LitWord:
		switch val {
		case "{", "if", "while", "until", "for", "case", "[[",
			"coproc", "let", "function", "declare", "local",
			"export", "readonly", "typeset", "nameref":
			return true
		}
	}
	return false
}

func (p *parser) coprocClause() *CoprocClause {
	cc := &CoprocClause{Coproc: p.pos}
	if p.next(); isBashCompoundCommand(p.tok, p.val) {
		// has no name
		cc.Stmt, _ = p.getStmt(false, false)
		return cc
	}
	if p.newLine {
		p.posErr(cc.Coproc, "coproc clause requires a command")
	}
	cc.Name = p.getLit()
	cc.Stmt, _ = p.getStmt(false, false)
	if cc.Stmt == nil {
		if cc.Name == nil {
			p.posErr(cc.Coproc, "coproc clause requires a command")
			return nil
		}
		// name was in fact the stmt
		cc.Stmt = p.stmt(cc.Name.ValuePos)
		cc.Stmt.Cmd = p.call(p.word(p.wps(cc.Name)))
		cc.Name = nil
	} else if cc.Name != nil {
		if call, ok := cc.Stmt.Cmd.(*CallExpr); ok {
			// name was in fact the start of a call
			call.Args = append([]*Word{p.word(p.wps(cc.Name))},
				call.Args...)
			cc.Name = nil
		}
	}
	return cc
}

func (p *parser) letClause() *LetClause {
	lc := &LetClause{Let: p.pos}
	old := p.preNested(arithmExprLet)
	p.next()
	for !p.newLine && !stopToken(p.tok) && !p.peekRedir() {
		x := p.arithmExpr(illegalTok, lc.Let, 0, true, false)
		if x == nil {
			break
		}
		lc.Exprs = append(lc.Exprs, x)
	}
	if len(lc.Exprs) == 0 {
		p.posErr(lc.Let, "let clause requires at least one expression")
	}
	p.postNested(old)
	if p.tok == illegalTok {
		p.next()
	}
	return lc
}

func (p *parser) bashFuncDecl() *FuncDecl {
	fpos := p.pos
	p.next()
	if p.tok != _LitWord {
		if w := p.followWord("function", fpos); p.err == nil {
			p.posErr(w.Pos(), "invalid func name")
		}
	}
	name := p.lit(p.pos, p.val)
	if p.next(); p.gotSameLine(leftParen) {
		p.follow(name.ValuePos, "foo(", rightParen)
	}
	return p.funcDecl(name, fpos)
}

func (p *parser) callExpr(s *Stmt, w *Word) *CallExpr {
	ce := p.call(w)
	for !p.newLine {
		switch p.tok {
		case _EOF, semicolon, and, or, andAnd, orOr, pipeAll,
			dblSemicolon, semiFall, dblSemiFall:
			return ce
		case _LitWord:
			ce.Args = append(ce.Args, p.word(
				p.wps(p.lit(p.pos, p.val)),
			))
			p.next()
		case bckQuote:
			if p.quote == subCmdBckquo {
				return ce
			}
			fallthrough
		case _Lit, dollBrace, dollDblParen, dollParen, dollar, cmdIn, cmdOut,
			sglQuote, dollSglQuote, dblQuote, dollDblQuote, dollBrack,
			globQuest, globStar, globPlus, globAt, globExcl:
			ce.Args = append(ce.Args, p.word(p.wordParts()))
		case rdrOut, appOut, rdrIn, dplIn, dplOut, clbOut, rdrInOut,
			hdoc, dashHdoc, wordHdoc, rdrAll, appAll, _LitRedir:
			p.doRedirect(s)
		case dblLeftParen:
			p.curErr("%s can only be used to open an arithmetic cmd", p.tok)
		case rightParen:
			if p.quote == subCmd {
				return ce
			}
			fallthrough
		default:
			p.curErr("a command can only contain words and redirects")
		}
	}
	return ce
}

func (p *parser) funcDecl(name *Lit, pos Pos) *FuncDecl {
	fd := &FuncDecl{
		Position:  pos,
		BashStyle: pos != name.ValuePos,
		Name:      name,
	}
	if fd.Body, _ = p.getStmt(false, false); fd.Body == nil {
		p.followErr(fd.Pos(), "foo()", "a statement")
	}
	return fd
}
