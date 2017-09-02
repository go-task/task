// Copyright (c) 2016, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package syntax

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"
)

// KeepComments makes the parser parse comments and attach them to
// nodes, as opposed to discarding them.
func KeepComments(p *Parser) { p.keepComments = true }

type LangVariant int

const (
	LangBash LangVariant = iota
	LangPOSIX
	LangMirBSDKorn
)

// Variant changes the shell language variant that the parser will
// accept.
func Variant(l LangVariant) func(*Parser) {
	return func(p *Parser) { p.lang = l }
}

// NewParser allocates a new Parser and applies any number of options.
func NewParser(options ...func(*Parser)) *Parser {
	p := &Parser{helperBuf: new(bytes.Buffer)}
	for _, opt := range options {
		opt(p)
	}
	return p
}

// Parse reads and parses a shell program with an optional name. It
// returns the parsed program if no issues were encountered. Otherwise,
// an error is returned. Reads from r are buffered.
//
// Parse can be called more than once, but not concurrently. That is, a
// Parser can be reused once it is done working.
func (p *Parser) Parse(r io.Reader, name string) (*File, error) {
	p.reset()
	p.f = &File{Name: name}
	p.src = r
	p.rune()
	p.next()
	p.f.StmtList = p.stmts()
	if p.err == nil {
		// EOF immediately after heredoc word so no newline to
		// trigger it
		p.doHeredocs()
	}
	return p.f, p.err
}

// Parser holds the internal state of the parsing mechanism of a
// program.
type Parser struct {
	src io.Reader
	bs  []byte // current chunk of read bytes
	bsp int    // pos within chunk for the rune after r
	r   rune   // next rune
	w   uint16 // width of r

	f *File

	spaced  bool // whether tok has whitespace on its left
	newLine bool // whether tok is on a new line

	err     error // lexer/parser error
	readErr error // got a read error, but bytes left

	tok token  // current token
	val string // current value (valid if tok is _Lit*)

	offs int
	pos  Pos // position of tok
	npos Pos // next position (of r)

	quote   quoteState // current lexer state
	eqlOffs int        // position of '=' in val (a literal)

	keepComments bool
	lang         LangVariant

	forbidNested bool

	// list of pending heredoc bodies
	buriedHdocs int
	heredocs    []*Redirect
	hdocStop    []byte

	accComs []Comment
	curComs *[]Comment

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

func (p *Parser) reset() {
	p.tok, p.val = illegalTok, ""
	p.eqlOffs = 0
	p.bs, p.bsp = nil, 0
	p.offs = 0
	p.npos = Pos{line: 1, col: 1}
	p.r, p.w = 0, 0
	p.err, p.readErr = nil, nil
	p.quote, p.forbidNested = noState, false
	p.heredocs, p.buriedHdocs = p.heredocs[:0], 0
	p.accComs, p.curComs = nil, &p.accComs
}

func (p *Parser) getPos() Pos {
	p.npos.offs = uint32(p.offs + p.bsp - int(p.w))
	return p.npos
}

func (p *Parser) lit(pos Pos, val string) *Lit {
	if len(p.litBatch) == 0 {
		p.litBatch = make([]Lit, 128)
	}
	l := &p.litBatch[0]
	p.litBatch = p.litBatch[1:]
	l.ValuePos = pos
	l.ValueEnd = p.getPos()
	l.Value = val
	return l
}

func (p *Parser) word(parts []WordPart) *Word {
	if len(p.wordBatch) == 0 {
		p.wordBatch = make([]Word, 64)
	}
	w := &p.wordBatch[0]
	p.wordBatch = p.wordBatch[1:]
	w.Parts = parts
	return w
}

func (p *Parser) wps(wp WordPart) []WordPart {
	if len(p.wpsBatch) == 0 {
		p.wpsBatch = make([]WordPart, 64)
	}
	wps := p.wpsBatch[:1:1]
	p.wpsBatch = p.wpsBatch[1:]
	wps[0] = wp
	return wps
}

func (p *Parser) stmt(pos Pos) *Stmt {
	if len(p.stmtBatch) == 0 {
		p.stmtBatch = make([]Stmt, 64)
	}
	s := &p.stmtBatch[0]
	p.stmtBatch = p.stmtBatch[1:]
	s.Position = pos
	return s
}

func (p *Parser) stList() []*Stmt {
	if len(p.stListBatch) == 0 {
		p.stListBatch = make([]*Stmt, 256)
	}
	stmts := p.stListBatch[:0:4]
	p.stListBatch = p.stListBatch[4:]
	return stmts
}

type callAlloc struct {
	ce CallExpr
	ws [4]*Word
}

func (p *Parser) call(w *Word) *CallExpr {
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

type quoteState uint32

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
	arrayElems

	allKeepSpaces = paramExpRepl | dblQuotes | hdocBody |
		hdocBodyTabs | paramExpExp | sglQuotes
	allRegTokens = noState | subCmd | subCmdBckquo | hdocWord |
		switchCase | arrayElems
	allArithmExpr = arithmExpr | arithmExprLet | arithmExprCmd |
		arithmExprBrack | allParamArith
	allRbrack     = arithmExprBrack | paramExpInd | paramName
	allParamArith = paramExpInd | paramExpOff | paramExpLen
	allParamReg   = paramName | paramExpName | allParamArith
	allParamExp   = allParamReg | paramExpRepl | paramExpExp
)

type saveState struct {
	quote       quoteState
	buriedHdocs int
}

func (p *Parser) preNested(quote quoteState) (s saveState) {
	s.quote, s.buriedHdocs = p.quote, p.buriedHdocs
	p.buriedHdocs, p.quote = len(p.heredocs), quote
	return
}

func (p *Parser) postNested(s saveState) {
	p.quote, p.buriedHdocs = s.quote, s.buriedHdocs
}

func (p *Parser) unquotedWordBytes(w *Word) ([]byte, bool) {
	p.helperBuf.Reset()
	didUnquote := false
	for _, wp := range w.Parts {
		if p.unquotedWordPart(p.helperBuf, wp, false) {
			didUnquote = true
		}
	}
	return p.helperBuf.Bytes(), didUnquote
}

func (p *Parser) unquotedWordPart(buf *bytes.Buffer, wp WordPart, quotes bool) (quoted bool) {
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

func (p *Parser) doHeredocs() {
	old := p.quote
	hdocs := p.heredocs[p.buriedHdocs:]
	p.heredocs = p.heredocs[:p.buriedHdocs]
	for i, r := range hdocs {
		if p.err != nil {
			break
		}
		p.quote = hdocBody
		if r.Op == DashHdoc {
			p.quote = hdocBodyTabs
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
			r.Hdoc = p.getWord()
		}
		if p.hdocStop != nil {
			p.posErr(r.Pos(), "unclosed here-document '%s'",
				string(p.hdocStop))
		}
	}
	p.quote = old
}

func (p *Parser) got(tok token) bool {
	if p.tok == tok {
		p.next()
		return true
	}
	return false
}

func (p *Parser) gotRsrv(val string) bool {
	if p.tok == _LitWord && p.val == val {
		p.next()
		return true
	}
	return false
}

func (p *Parser) gotSameLine(tok token) bool {
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

func (p *Parser) followErr(pos Pos, left, right string) {
	leftStr := readableStr(left)
	p.posErr(pos, "%s must be followed by %s", leftStr, right)
}

func (p *Parser) followErrExp(pos Pos, left string) {
	p.followErr(pos, left, "an expression")
}

func (p *Parser) follow(lpos Pos, left string, tok token) {
	if !p.got(tok) {
		p.followErr(lpos, left, tok.String())
	}
}

func (p *Parser) followRsrv(lpos Pos, left, val string) Pos {
	pos := p.pos
	if !p.gotRsrv(val) {
		p.followErr(lpos, left, fmt.Sprintf("%q", val))
	}
	return pos
}

func (p *Parser) followStmts(left string, lpos Pos, stops ...string) StmtList {
	if p.gotSameLine(semicolon) {
		return StmtList{}
	}
	sl := p.stmts(stops...)
	if len(sl.Stmts) < 1 && !p.newLine {
		p.followErr(lpos, left, "a statement list")
	}
	return sl
}

func (p *Parser) followWordTok(tok token, pos Pos) *Word {
	w := p.getWord()
	if w == nil {
		p.followErr(pos, tok.String(), "a word")
	}
	return w
}

func (p *Parser) followWord(s string, pos Pos) *Word {
	w := p.getWord()
	if w == nil {
		p.followErr(pos, s, "a word")
	}
	return w
}

func (p *Parser) stmtEnd(n Node, start, end string) Pos {
	pos := p.pos
	if !p.gotRsrv(end) {
		p.posErr(n.Pos(), "%s statement must end with %q", start, end)
	}
	return pos
}

func (p *Parser) quoteErr(lpos Pos, quote token) {
	p.posErr(lpos, "reached %s without closing quote %s",
		p.tok.String(), quote)
}

func (p *Parser) matchingErr(lpos Pos, left, right interface{}) {
	p.posErr(lpos, "reached %s without matching %s with %s",
		p.tok.String(), left, right)
}

func (p *Parser) matched(lpos Pos, left, right token) Pos {
	pos := p.pos
	if !p.got(right) {
		p.matchingErr(lpos, left, right)
	}
	return pos
}

func (p *Parser) errPass(err error) {
	if p.err == nil {
		p.err = err
		p.bsp = len(p.bs) + 1
		p.r = utf8.RuneSelf
		p.tok = _EOF
	}
}

// ParseError represents an error found when parsing a source file.
type ParseError struct {
	Filename string
	Pos
	Text string
}

func (e *ParseError) Error() string {
	if e.Filename == "" {
		return fmt.Sprintf("%s: %s", e.Pos.String(), e.Text)
	}
	return fmt.Sprintf("%s:%s: %s", e.Filename, e.Pos.String(), e.Text)
}

func (p *Parser) posErr(pos Pos, format string, a ...interface{}) {
	p.errPass(&ParseError{
		Filename: p.f.Name,
		Pos:      pos,
		Text:     fmt.Sprintf(format, a...),
	})
}

func (p *Parser) curErr(format string, a ...interface{}) {
	p.posErr(p.pos, format, a...)
}

func (p *Parser) stmts(stops ...string) (sl StmtList) {
	gotEnd := true
loop:
	for p.tok != _EOF {
		switch p.tok {
		case _LitWord:
			for _, stop := range stops {
				if p.val == stop {
					break loop
				}
			}
		case rightParen:
			if p.quote == subCmd {
				break loop
			}
		case bckQuote:
			if p.quote == subCmdBckquo {
				break loop
			}
		case dblSemicolon, semiAnd, dblSemiAnd, semiOr:
			if p.quote == switchCase {
				break loop
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
			if sl.Stmts == nil {
				sl.Stmts = p.stList()
			}
			sl.Stmts = append(sl.Stmts, s)
			gotEnd = end
		}
	}
	sl.Last, p.accComs = p.accComs, nil
	return
}

func (p *Parser) invalidStmtStart() {
	switch p.tok {
	case semicolon, and, or, andAnd, orOr:
		p.curErr("%s can only immediately follow a statement", p.tok)
	case rightParen:
		p.curErr("%s can only be used to close a subshell", p.tok)
	default:
		p.curErr("%s is not a valid start for a statement", p.tok)
	}
}

func (p *Parser) getWord() *Word {
	if parts := p.wordParts(); len(parts) > 0 {
		return p.word(parts)
	}
	return nil
}

func (p *Parser) getLit() *Lit {
	switch p.tok {
	case _Lit, _LitWord, _LitRedir:
		l := p.lit(p.pos, p.val)
		p.next()
		return l
	}
	return nil
}

func (p *Parser) wordParts() (wps []WordPart) {
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

func (p *Parser) ensureNoNested() {
	if p.forbidNested {
		p.curErr("expansions not allowed in heredoc words")
	}
}

func (p *Parser) wordPart() WordPart {
	switch p.tok {
	case _Lit, _LitWord:
		l := p.lit(p.pos, p.val)
		p.next()
		return l
	case dollBrace:
		p.ensureNoNested()
		switch p.r {
		case '|':
			if p.lang != LangMirBSDKorn {
				p.curErr(`"${|stmts;}" is a mksh feature`)
			}
			fallthrough
		case ' ', '\t', '\n':
			if p.lang != LangMirBSDKorn {
				p.curErr(`"${ stmts;}" is a mksh feature`)
			}
			cs := &CmdSubst{
				Left:     p.pos,
				TempFile: p.r != '|',
				ReplyVar: p.r == '|',
			}
			old := p.preNested(subCmd)
			p.rune() // don't tokenize '|'
			p.next()
			cs.StmtList = p.stmts("}")
			p.postNested(old)
			cs.Right = p.pos
			if !p.gotRsrv("}") {
				p.matchingErr(cs.Left, "${", "}")
			}
			return cs
		default:
			return p.paramExp()
		}
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
		if p.got(hash) {
			if p.lang != LangMirBSDKorn {
				p.posErr(ar.Pos(), "unsigned expressions are a mksh feature")
			}
			ar.Unsigned = true
		}
		ar.X = p.followArithm(left, ar.Left)
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
		cs.StmtList = p.stmts()
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
		return p.shortParamExp()
	case cmdIn, cmdOut:
		p.ensureNoNested()
		ps := &ProcSubst{Op: ProcOperator(p.tok), OpPos: p.pos}
		old := p.preNested(subCmd)
		p.next()
		ps.StmtList = p.stmts()
		p.postNested(old)
		ps.Rparen = p.matched(ps.OpPos, token(ps.Op), rightParen)
		return ps
	case sglQuote:
		if p.quote&allArithmExpr != 0 {
			p.curErr("quotes should not be used in arithmetic expressions")
		}
		sq := &SglQuoted{Left: p.pos}
		r := p.r
	loop:
		for p.newLit(r); ; r = p.rune() {
			switch r {
			case utf8.RuneSelf, '\'':
				sq.Right = p.getPos()
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
		if p.quote&allArithmExpr != 0 {
			p.curErr("quotes should not be used in arithmetic expressions")
		}
		sq := &SglQuoted{Left: p.pos, Dollar: true}
		old := p.quote
		p.quote = sglQuotes
		p.next()
		p.quote = old
		if p.tok != sglQuote {
			sq.Value = p.val
			p.next()
		}
		sq.Right = p.pos
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
		if p.quote&allArithmExpr != 0 {
			p.curErr("quotes should not be used in arithmetic expressions")
		}
		return p.dblQuoted()
	case bckQuote:
		if p.quote == subCmdBckquo {
			return nil
		}
		p.ensureNoNested()
		cs := &CmdSubst{Left: p.pos}
		old := p.preNested(subCmdBckquo)
		p.next()
		cs.StmtList = p.stmts()
		p.postNested(old)
		cs.Right = p.pos
		if !p.got(bckQuote) {
			p.quoteErr(cs.Pos(), bckQuote)
		}
		return cs
	case globQuest, globStar, globPlus, globAt, globExcl:
		if p.lang == LangPOSIX {
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
		eg.Pattern = p.lit(posAddCol(eg.OpPos, 2), p.endLit())
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

func (p *Parser) dblQuoted() *DblQuoted {
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

func (p *Parser) followArithm(ftok token, fpos Pos) ArithmExpr {
	x := p.arithmExpr(0, false, false)
	if x == nil {
		p.followErrExp(fpos, ftok.String())
	}
	return x
}

func (p *Parser) arithmExpr(level int, compact, tern bool) ArithmExpr {
	if p.tok == _EOF || p.peekArithmEnd() {
		return nil
	}
	var left ArithmExpr
	if level > 11 {
		left = p.arithmExprBase(compact)
	} else {
		left = p.arithmExpr(level+1, compact, false)
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
		case leftBrack:
			p.curErr("[ must follow a name")
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
		if !isArithName(b.X) {
			p.posErr(b.OpPos, "%s must follow a name", b.Op.String())
		}
	}
	if p.next(); compact && p.spaced {
		p.followErrExp(b.OpPos, b.Op.String())
	}
	b.Y = p.arithmExpr(newLevel, compact, b.Op == Quest)
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

func isArithName(left ArithmExpr) bool {
	w, ok := left.(*Word)
	if !ok || len(w.Parts) != 1 {
		return false
	}
	switch x := w.Parts[0].(type) {
	case *Lit:
		return ValidName(x.Value)
	case *ParamExp:
		return x.nakedIndex()
	default:
		return false
	}
}

func (p *Parser) arithmExprBase(compact bool) ArithmExpr {
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
		if p.tok != _LitWord {
			p.followErr(ue.OpPos, token(ue.Op).String(), "a literal")
		}
		ue.X = p.arithmExprBase(compact)
		return ue
	case leftParen:
		pe := &ParenArithm{Lparen: p.pos}
		p.next()
		pe.X = p.followArithm(leftParen, pe.Lparen)
		pe.Rparen = p.matched(pe.Lparen, leftParen, rightParen)
		x = pe
	case plus, minus:
		ue := &UnaryArithm{OpPos: p.pos, Op: UnAritOperator(p.tok)}
		if p.next(); compact && p.spaced {
			p.followErrExp(ue.OpPos, ue.Op.String())
		}
		ue.X = p.arithmExprBase(compact)
		if ue.X == nil {
			p.followErrExp(ue.OpPos, ue.Op.String())
		}
		x = ue
	case _LitWord:
		l := p.getLit()
		if p.tok != leftBrack {
			x = p.word(p.wps(l))
			break
		}
		left := p.pos
		pe := &ParamExp{Dollar: l.ValuePos, Short: true, Param: l}
		old := p.preNested(arithmExprBrack)
		p.next()
		if p.tok == dblQuote {
			pe.Index = p.word(p.wps(p.dblQuoted()))
		} else {
			pe.Index = p.followArithm(leftBrack, left)
		}
		p.postNested(old)
		p.matched(left, leftBrack, rightBrack)
		x = p.word(p.wps(pe))
	case bckQuote:
		if p.quote == arithmExprLet {
			return nil
		}
		fallthrough
	default:
		if w := p.getWord(); w != nil {
			// we want real nil, not (*Word)(nil) as that
			// sets the type to non-nil and then x != nil
			x = w
		}
	}
	if compact && p.spaced {
		return x
	}
	if p.tok == addAdd || p.tok == subSub {
		if !isArithName(x) {
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

func (p *Parser) shortParamExp() *ParamExp {
	pe := &ParamExp{Dollar: p.pos, Short: true}
	p.pos = posAddCol(p.pos, 1)
	switch p.r {
	case '@', '*', '#', '$', '?', '!', '0', '-':
		p.tok, p.val = _LitWord, string(p.r)
		p.rune()
	default:
		old := p.quote
		p.quote = paramName
		p.advanceLitOther(p.r)
		p.quote = old
	}
	pe.Param = p.getLit()
	return pe
}

func (p *Parser) paramExp() *ParamExp {
	pe := &ParamExp{Dollar: p.pos}
	old := p.quote
	p.quote = paramExpName
	if p.r == '#' {
		p.tok = hash
		p.pos = p.getPos()
		p.rune()
	} else {
		p.next()
	}
	switch p.tok {
	case at:
		p.tok, p.val = _LitWord, "@"
	case hash:
		if paramNameOp(p.r) {
			pe.Length = true
			p.next()
		}
	case perc:
		if p.lang != LangMirBSDKorn {
			p.posErr(pe.Pos(), `"${%%foo}" is a mksh feature`)
		}
		if paramNameOp(p.r) {
			pe.Width = true
			p.next()
		}
	case exclMark:
		if paramNameOp(p.r) {
			pe.Excl = true
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
		p.curErr("parameter expansion requires a literal")
	}
	if p.tok == rightBrace {
		pe.Rbrace = p.pos
		p.quote = old
		p.next()
		return pe
	}
	if p.tok == leftBrack {
		if p.lang == LangPOSIX {
			p.curErr("arrays are a bash feature")
		}
		lpos := p.pos
		p.quote = paramExpInd
		p.next()
		switch p.tok {
		case star, at:
			p.tok, p.val = _LitWord, p.tok.String()
		}
		if p.tok == dblQuote {
			pe.Index = p.word(p.wps(p.dblQuoted()))
		} else {
			pe.Index = p.followArithm(leftBrack, lpos)
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
		if p.lang == LangPOSIX {
			p.curErr("search and replace is a bash feature")
		}
		pe.Repl = &Replace{All: p.tok == dblSlash}
		p.quote = paramExpRepl
		p.next()
		pe.Repl.Orig = p.getWord()
		p.quote = paramExpExp
		if p.got(slash) {
			pe.Repl.With = p.getWord()
		}
	case colon:
		if p.lang == LangPOSIX {
			p.curErr("slicing is a bash feature")
		}
		pe.Slice = &Slice{}
		colonPos := p.pos
		p.quote = paramExpOff
		if p.next(); p.tok != colon {
			pe.Slice.Offset = p.followArithm(colon, colonPos)
		}
		colonPos = p.pos
		p.quote = paramExpLen
		if p.got(colon) {
			pe.Slice.Length = p.followArithm(colon, colonPos)
		}
	case caret, dblCaret, comma, dblComma:
		if p.lang != LangBash {
			p.curErr("this expansion operator is a bash feature")
		}
		fallthrough
	case at:
		if p.lang == LangPOSIX {
			p.curErr("this expansion operator is a bash feature")
		}
		fallthrough
	default:
		pe.Exp = &Expansion{Op: ParExpOperator(p.tok)}
		p.quote = paramExpExp
		p.next()
		pe.Exp.Word = p.getWord()
	}
	p.quote = old
	pe.Rbrace = p.pos
	p.matched(pe.Dollar, dollBrace, rightBrace)
	return pe
}

func (p *Parser) peekArithmEnd() bool {
	return p.tok == rightParen && p.r == ')'
}

func (p *Parser) arithmEnd(ltok token, lpos Pos, old saveState) Pos {
	if !p.peekArithmEnd() {
		p.matchingErr(lpos, ltok, dblRightParen)
	}
	p.rune()
	p.postNested(old)
	pos := p.pos
	p.next()
	return pos
}

func stopToken(tok token) bool {
	switch tok {
	case _EOF, semicolon, and, or, andAnd, orOr, orAnd, dblSemicolon,
		semiAnd, dblSemiAnd, semiOr, rightParen:
		return true
	}
	return false
}

// ValidName returns whether val is a valid name as per the POSIX spec.
func ValidName(val string) bool {
	for i, c := range val {
		switch {
		case 'a' <= c && c <= 'z':
		case 'A' <= c && c <= 'Z':
		case c == '_':
		case i > 0 && '0' <= c && c <= '9':
		default:
			return false
		}
	}
	return true
}

func (p *Parser) hasValidIdent() bool {
	if end := p.eqlOffs; end > 0 {
		if p.val[end-1] == '+' && p.lang != LangPOSIX {
			end--
		}
		if ValidName(p.val[:end]) {
			return true
		}
	}
	return p.tok == _Lit && p.r == '['
}

func (p *Parser) getAssign(needEqual bool) *Assign {
	as := &Assign{}
	if p.eqlOffs > 0 { // foo=bar
		nameEnd := p.eqlOffs
		if p.lang != LangPOSIX && p.val[p.eqlOffs-1] == '+' {
			// a+=b
			as.Append = true
			nameEnd--
		}
		as.Name = p.lit(p.pos, p.val[:nameEnd])
		// since we're not using the entire p.val
		as.Name.ValueEnd = posAddCol(as.Name.ValuePos, nameEnd)
		left := p.lit(posAddCol(p.pos, 1), p.val[p.eqlOffs+1:])
		if left.Value != "" {
			left.ValuePos = posAddCol(left.ValuePos, p.eqlOffs)
			as.Value = p.word(p.wps(left))
		}
		p.next()
	} else { // foo[x]=bar
		as.Name = p.lit(p.pos, p.val)
		// hasValidIdent already checks p.r is '['
		p.rune()
		left := posAddCol(p.pos, 1)
		old := p.preNested(arithmExprBrack)
		p.next()
		if p.tok == star {
			p.tok, p.val = _LitWord, p.tok.String()
		}
		if p.tok == dblQuote {
			as.Index = p.word(p.wps(p.dblQuoted()))
		} else {
			as.Index = p.followArithm(leftBrack, left)
		}
		p.postNested(old)
		p.matched(left, leftBrack, rightBrack)
		if !needEqual && (p.spaced || stopToken(p.tok)) {
			return as
		}
		if len(p.val) > 0 && p.val[0] == '+' {
			as.Append = true
			p.val = p.val[1:]
			p.pos = posAddCol(p.pos, 1)
		}
		if len(p.val) < 1 || p.val[0] != '=' {
			if as.Append {
				p.followErr(as.Pos(), "a[b]+", "=")
			} else {
				p.followErr(as.Pos(), "a[b]", "=")
			}
			return nil
		}
		p.pos = posAddCol(p.pos, 1)
		p.val = p.val[1:]
		if p.val == "" {
			p.next()
		}
	}
	if p.spaced || stopToken(p.tok) {
		return as
	}
	if as.Value == nil && p.tok == leftParen {
		if p.lang == LangPOSIX {
			p.curErr("arrays are a bash feature")
		}
		as.Array = &ArrayExpr{Lparen: p.pos}
		newQuote := p.quote
		if p.lang == LangBash {
			newQuote = arrayElems
		}
		old := p.preNested(newQuote)
		p.next()
		for p.tok != _EOF && p.tok != rightParen {
			ae := &ArrayElem{}
			ae.Comments, p.accComs = p.accComs, nil
			if p.tok == leftBrack {
				left := p.pos
				p.quote = arithmExprBrack
				p.next()
				if p.tok == dblQuote {
					ae.Index = p.word(p.wps(p.dblQuoted()))
				} else {
					ae.Index = p.followArithm(leftBrack, left)
				}
				if p.tok != rightBrack {
					p.matchingErr(left, leftBrack, rightBrack)
				}
				p.quote = arrayElems
				if p.r != '=' {
					p.followErr(left, `"[x]"`, "=")
				}
				p.rune()
				p.next()
			}
			if ae.Value = p.getWord(); ae.Value == nil {
				p.curErr("array element values must be words")
				break
			}
			if len(p.accComs) > 0 {
				c := p.accComs[0]
				if c.Pos().Line() == ae.End().Line() {
					ae.Comments = append(ae.Comments, c)
					p.accComs = p.accComs[1:]
				}
			}
			as.Array.Elems = append(as.Array.Elems, ae)
		}
		as.Array.Last, p.accComs = p.accComs, nil
		p.postNested(old)
		as.Array.Rparen = p.matched(as.Array.Lparen, leftParen, rightParen)
	} else if w := p.getWord(); w != nil {
		if as.Value == nil {
			as.Value = w
		} else {
			as.Value.Parts = append(as.Value.Parts, w.Parts...)
		}
	}
	return as
}

func (p *Parser) peekRedir() bool {
	switch p.tok {
	case rdrOut, appOut, rdrIn, dplIn, dplOut, clbOut, rdrInOut,
		hdoc, dashHdoc, wordHdoc, rdrAll, appAll, _LitRedir:
		return true
	}
	return false
}

func (p *Parser) doRedirect(s *Stmt) {
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

func (p *Parser) getStmt(readEnd, binCmd bool) (s *Stmt, gotEnd bool) {
	s = p.stmt(p.pos)
	if p.gotRsrv("!") {
		s.Negated = true
		if p.newLine || stopToken(p.tok) {
			p.posErr(s.Pos(), `"!" cannot form a statement alone`)
		}
	}
	if s = p.gotStmtPipe(s); s == nil || p.err != nil {
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
			if b.Y, _ = p.getStmt(false, true); b.Y == nil || p.err != nil {
				p.followErr(b.OpPos, b.Op.String(), "a statement")
				return
			}
			s = p.stmt(s.Position)
			s.Cmd = b
			s.Comments, b.X.Comments = b.X.Comments, nil
		}
		if p.tok != semicolon {
			break
		}
		fallthrough
	case semicolon:
		if !p.newLine && readEnd {
			s.Semicolon = p.pos
			p.next()
		}
	case and:
		p.next()
		s.Background = true
	case orAnd:
		p.next()
		s.Coprocess = true
	}
	gotEnd = s.Semicolon.IsValid() || s.Background || s.Coprocess
	if len(p.accComs) > 0 && !binCmd {
		c := p.accComs[0]
		if c.Pos().Line() == s.End().Line() {
			s.Comments = append(s.Comments, c)
			p.accComs = p.accComs[1:]
		}
	}
	return
}

func (p *Parser) gotStmtPipe(s *Stmt) *Stmt {
	s.Comments, p.accComs = p.accComs, nil
	switch p.tok {
	case _LitWord:
		switch p.val {
		case "{":
			s.Cmd = p.block()
		case "if":
			s.Cmd = p.ifClause()
		case "while", "until":
			s.Cmd = p.whileClause(p.val == "until")
		case "for":
			s.Cmd = p.forClause()
		case "case":
			s.Cmd = p.caseClause()
		case "}":
			p.curErr(`%q can only be used to close a block`, p.val)
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
		case "!":
			if !s.Negated {
				p.curErr(`"!" can only be used in full statements`)
				break
			}
		case "[[":
			if p.lang != LangPOSIX {
				s.Cmd = p.testClause()
			}
		case "]]":
			if p.lang != LangPOSIX {
				p.curErr(`%q can only be used to close a test`,
					p.val)
			}
		case "let":
			if p.lang != LangPOSIX {
				s.Cmd = p.letClause()
			}
		case "function":
			if p.lang != LangPOSIX {
				s.Cmd = p.bashFuncDecl()
			}
		case "declare":
			if p.lang == LangBash {
				s.Cmd = p.declClause()
			}
		case "local", "export", "readonly", "typeset", "nameref":
			if p.lang != LangPOSIX {
				s.Cmd = p.declClause()
			}
		case "time":
			if p.lang != LangPOSIX {
				s.Cmd = p.timeClause()
			}
		case "coproc":
			if p.lang == LangBash {
				s.Cmd = p.coprocClause()
			}
		case "select":
			if p.lang != LangPOSIX {
				s.Cmd = p.selectClause()
			}
		}
		if s.Cmd != nil {
			break
		}
		if p.hasValidIdent() {
			s.Cmd = p.callExpr(s, nil, true)
			break
		}
		name := p.lit(p.pos, p.val)
		if p.next(); p.gotSameLine(leftParen) {
			p.follow(name.ValuePos, "foo(", rightParen)
			if p.lang == LangPOSIX && !ValidName(name.Value) {
				p.posErr(name.Pos(), "invalid func name")
			}
			s.Cmd = p.funcDecl(name, name.ValuePos)
		} else {
			s.Cmd = p.callExpr(s, p.word(p.wps(name)), false)
		}
	case rdrOut, appOut, rdrIn, dplIn, dplOut, clbOut, rdrInOut,
		hdoc, dashHdoc, wordHdoc, rdrAll, appAll, _LitRedir:
		p.doRedirect(s)
		switch {
		case p.newLine, p.tok == _EOF, p.tok == semicolon:
			return s
		}
		s.Cmd = p.callExpr(s, nil, false)
	case bckQuote:
		if p.quote == subCmdBckquo {
			return nil
		}
		fallthrough
	case _Lit, dollBrace, dollDblParen, dollParen, dollar, cmdIn, cmdOut,
		sglQuote, dollSglQuote, dblQuote, dollDblQuote, dollBrack,
		globQuest, globStar, globPlus, globAt, globExcl:
		if p.tok == _Lit && p.hasValidIdent() {
			s.Cmd = p.callExpr(s, nil, true)
			break
		}
		w := p.word(p.wordParts())
		if p.gotSameLine(leftParen) && p.err == nil {
			p.posErr(w.Pos(), "invalid func name")
		}
		s.Cmd = p.callExpr(s, w, false)
	case leftParen:
		s.Cmd = p.subshell()
	case dblLeftParen:
		s.Cmd = p.arithmExpCmd()
	default:
		if len(s.Redirs) == 0 {
			return nil
		}
	}
	for !p.newLine && p.peekRedir() {
		p.doRedirect(s)
	}
	switch p.tok {
	case orAnd:
		if p.lang == LangMirBSDKorn {
			break
		}
		fallthrough
	case or:
		b := &BinaryCmd{OpPos: p.pos, Op: BinCmdOperator(p.tok), X: s}
		p.next()
		if b.Y = p.gotStmtPipe(p.stmt(p.pos)); b.Y == nil || p.err != nil {
			p.followErr(b.OpPos, b.Op.String(), "a statement")
			break
		}
		s = p.stmt(s.Position)
		s.Cmd = b
		s.Comments, b.X.Comments = b.X.Comments, nil
	}
	return s
}

func (p *Parser) subshell() *Subshell {
	s := &Subshell{Lparen: p.pos}
	old := p.preNested(subCmd)
	p.next()
	s.StmtList = p.stmts()
	p.postNested(old)
	s.Rparen = p.matched(s.Lparen, leftParen, rightParen)
	return s
}

func (p *Parser) arithmExpCmd() Command {
	ar := &ArithmCmd{Left: p.pos}
	old := p.preNested(arithmExprCmd)
	p.next()
	if p.got(hash) {
		if p.lang != LangMirBSDKorn {
			p.posErr(ar.Pos(), "unsigned expressions are a mksh feature")
		}
		ar.Unsigned = true
	}
	ar.X = p.followArithm(dblLeftParen, ar.Left)
	ar.Right = p.arithmEnd(dblLeftParen, ar.Left, old)
	return ar
}

func (p *Parser) block() *Block {
	b := &Block{Lbrace: p.pos}
	p.next()
	b.StmtList = p.stmts("}")
	b.Rbrace = p.pos
	if !p.gotRsrv("}") {
		p.matchingErr(b.Lbrace, "{", "}")
	}
	return b
}

func (p *Parser) ifClause() *IfClause {
	rif := &IfClause{IfPos: p.pos}
	p.next()
	rif.Cond = p.followStmts("if", rif.IfPos, "then")
	rif.ThenPos = p.followRsrv(rif.IfPos, "if <cond>", "then")
	rif.Then = p.followStmts("then", rif.ThenPos, "fi", "elif", "else")
	curIf := rif
	for p.tok == _LitWord && p.val == "elif" {
		elf := &IfClause{IfPos: p.pos, Elif: true}
		p.next()
		elf.Cond = p.followStmts("elif", elf.IfPos, "then")
		elf.ThenPos = p.followRsrv(elf.IfPos, "elif <cond>", "then")
		elf.Then = p.followStmts("then", elf.ThenPos, "fi", "elif", "else")
		s := p.stmt(elf.IfPos)
		s.Cmd = elf
		curIf.ElsePos = elf.IfPos
		curIf.Else.Stmts = []*Stmt{s}
		curIf = elf
	}
	if elsePos := p.pos; p.gotRsrv("else") {
		curIf.ElsePos = elsePos
		curIf.Else = p.followStmts("else", curIf.ElsePos, "fi")
	}
	rif.FiPos = p.stmtEnd(rif, "if", "fi")
	curIf.FiPos = rif.FiPos
	return rif
}

func (p *Parser) whileClause(until bool) *WhileClause {
	wc := &WhileClause{WhilePos: p.pos, Until: until}
	rsrv := "while"
	rsrvCond := "while <cond>"
	if wc.Until {
		rsrv = "until"
		rsrvCond = "until <cond>"
	}
	p.next()
	wc.Cond = p.followStmts(rsrv, wc.WhilePos, "do")
	wc.DoPos = p.followRsrv(wc.WhilePos, rsrvCond, "do")
	wc.Do = p.followStmts("do", wc.DoPos, "done")
	wc.DonePos = p.stmtEnd(wc, rsrv, "done")
	return wc
}

func (p *Parser) forClause() *ForClause {
	fc := &ForClause{ForPos: p.pos}
	p.next()
	fc.Loop = p.loop(fc.ForPos)
	fc.DoPos = p.followRsrv(fc.ForPos, "for foo [in words]", "do")
	fc.Do = p.followStmts("do", fc.DoPos, "done")
	fc.DonePos = p.stmtEnd(fc, "for", "done")
	return fc
}

func (p *Parser) loop(fpos Pos) Loop {
	if p.lang != LangBash {
		switch p.tok {
		case leftParen, dblLeftParen:
			p.curErr("c-style fors are a bash feature")
		}
	}
	if p.tok == dblLeftParen {
		cl := &CStyleLoop{Lparen: p.pos}
		old := p.preNested(arithmExprCmd)
		p.next()
		cl.Init = p.arithmExpr(0, false, false)
		if !p.got(dblSemicolon) {
			p.follow(p.pos, "expr", semicolon)
			cl.Cond = p.arithmExpr(0, false, false)
			p.follow(p.pos, "expr", semicolon)
		}
		cl.Post = p.arithmExpr(0, false, false)
		cl.Rparen = p.arithmEnd(dblLeftParen, cl.Lparen, old)
		p.gotSameLine(semicolon)
		return cl
	}
	return p.wordIter("for", fpos)
}

func (p *Parser) wordIter(ftok string, fpos Pos) *WordIter {
	wi := &WordIter{}
	if wi.Name = p.getLit(); wi.Name == nil {
		p.followErr(fpos, ftok, "a literal")
	}
	if p.gotRsrv("in") {
		for !p.newLine && p.tok != _EOF && p.tok != semicolon {
			if w := p.getWord(); w == nil {
				p.curErr("word list can only contain words")
			} else {
				wi.Items = append(wi.Items, w)
			}
		}
		p.gotSameLine(semicolon)
	} else if !p.newLine && !p.got(semicolon) {
		p.followErr(fpos, ftok+" foo", `"in", ; or a newline`)
	}
	return wi
}

func (p *Parser) selectClause() *ForClause {
	fc := &ForClause{ForPos: p.pos, Select: true}
	p.next()
	fc.Loop = p.wordIter("select", fc.ForPos)
	fc.DoPos = p.followRsrv(fc.ForPos, "select foo [in words]", "do")
	fc.Do = p.followStmts("do", fc.DoPos, "done")
	fc.DonePos = p.stmtEnd(fc, "select", "done")
	return fc
}

func (p *Parser) caseClause() *CaseClause {
	cc := &CaseClause{Case: p.pos}
	p.next()
	cc.Word = p.followWord("case", cc.Case)
	end := "esac"
	if p.gotRsrv("{") {
		if p.lang != LangMirBSDKorn {
			p.posErr(cc.Pos(), `"case i {" is a mksh feature`)
		}
		end = "}"
	} else {
		p.followRsrv(cc.Case, "case x", "in")
	}
	cc.Items = p.caseItems(end)
	cc.Last, p.accComs = p.accComs, nil
	cc.Esac = p.stmtEnd(cc, "case", end)
	return cc
}

func (p *Parser) caseItems(stop string) (items []*CaseItem) {
	for p.tok != _EOF && !(p.tok == _LitWord && p.val == stop) {
		ci := &CaseItem{}
		ci.Comments, p.accComs = p.accComs, nil
		p.got(leftParen)
		for p.tok != _EOF {
			if w := p.getWord(); w == nil {
				p.curErr("case patterns must consist of words")
			} else {
				ci.Patterns = append(ci.Patterns, w)
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
		ci.StmtList = p.stmts(stop)
		p.postNested(old)
		ci.OpPos = p.pos
		switch p.tok {
		case dblSemicolon, semiAnd, dblSemiAnd, semiOr:
		default:
			ci.Op = Break
			items = append(items, ci)
			return
		}
		ci.Last = append(ci.Last, p.accComs...)
		p.accComs = nil
		ci.Op = CaseOperator(p.tok)
		p.next()
		if len(p.accComs) > 0 {
			c := p.accComs[0]
			if c.Pos().Line() == ci.OpPos.Line() {
				ci.Comments = append(ci.Comments, c)
				p.accComs = p.accComs[1:]
			}
		}
		items = append(items, ci)
	}
	return
}

func (p *Parser) testClause() *TestClause {
	tc := &TestClause{Left: p.pos}
	if p.next(); p.tok == _EOF || p.gotRsrv("]]") {
		p.posErr(tc.Left, "test clause requires at least one expression")
	}
	tc.X = p.testExpr(illegalTok, tc.Left, false)
	tc.Right = p.pos
	if !p.gotRsrv("]]") {
		p.matchingErr(tc.Left, "[[", "]]")
	}
	return tc
}

func (p *Parser) testExpr(ftok token, fpos Pos, pastAndOr bool) TestExpr {
	var left TestExpr
	if pastAndOr {
		left = p.testExprBase(ftok, fpos)
	} else {
		left = p.testExpr(ftok, fpos, true)
	}
	if left == nil {
		return left
	}
	switch p.tok {
	case andAnd, orOr:
	case _LitWord:
		if p.val == "]]" {
			return left
		}
	case rdrIn, rdrOut:
	case _EOF, rightParen:
		return left
	case _Lit:
		p.curErr("test operator words must consist of a single literal")
	default:
		p.curErr("not a valid test operator: %v", p.tok)
	}
	if p.tok == _LitWord {
		if p.tok = token(testBinaryOp(p.val)); p.tok == illegalTok {
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
		if b.Y = p.testExpr(token(b.Op), b.OpPos, false); b.Y == nil {
			p.followErrExp(b.OpPos, b.Op.String())
		}
	case TsReMatch:
		if p.lang != LangBash {
			p.curErr("regex tests are a bash feature")
		}
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

func (p *Parser) testExprBase(ftok token, fpos Pos) TestExpr {
	switch p.tok {
	case _EOF, rightParen:
		return nil
	case _LitWord:
		op := token(testUnaryOp(p.val))
		switch op {
		case illegalTok:
		case tsRefVar, tsModif: // not available in mksh
			if p.lang == LangBash {
				p.tok = op
			}
		default:
			p.tok = op
		}
	}
	switch p.tok {
	case exclMark:
		u := &UnaryTest{OpPos: p.pos, Op: TsNot}
		p.next()
		if u.X = p.testExpr(token(u.Op), u.OpPos, false); u.X == nil {
			p.followErrExp(u.OpPos, u.Op.String())
		}
		return u
	case tsExists, tsRegFile, tsDirect, tsCharSp, tsBlckSp, tsNmPipe,
		tsSocket, tsSmbLink, tsSticky, tsGIDSet, tsUIDSet, tsGrpOwn,
		tsUsrOwn, tsModif, tsRead, tsWrite, tsExec, tsNoEmpty,
		tsFdTerm, tsEmpStr, tsNempStr, tsOptSet, tsVarSet, tsRefVar:
		u := &UnaryTest{OpPos: p.pos, Op: UnTestOperator(p.tok)}
		p.next()
		u.X = p.followWordTok(token(u.Op), u.OpPos)
		return u
	case leftParen:
		pe := &ParenTest{Lparen: p.pos}
		p.next()
		if pe.X = p.testExpr(leftParen, pe.Lparen, false); pe.X == nil {
			p.followErrExp(pe.Lparen, "(")
		}
		pe.Rparen = p.matched(pe.Lparen, leftParen, rightParen)
		return pe
	default:
		// since we don't have [[ as a token
		fstr := "[["
		if ftok != illegalTok {
			fstr = ftok.String()
		}
		return p.followWord(fstr, fpos)
	}
}

func (p *Parser) declClause() *DeclClause {
	ds := &DeclClause{Variant: p.lit(p.pos, p.val)}
	p.next()
	for (p.tok == _LitWord || p.tok == _Lit) && p.val[0] == '-' {
		ds.Opts = append(ds.Opts, p.getWord())
	}
	for !p.newLine && !stopToken(p.tok) && !p.peekRedir() {
		if (p.tok == _Lit || p.tok == _LitWord) && p.hasValidIdent() {
			ds.Assigns = append(ds.Assigns, p.getAssign(false))
		} else if p.eqlOffs > 0 {
			p.curErr("invalid var name")
		} else if p.tok == _LitWord {
			ds.Assigns = append(ds.Assigns, &Assign{
				Naked: true,
				Name:  p.getLit(),
			})
		} else if w := p.getWord(); w != nil {
			ds.Assigns = append(ds.Assigns, &Assign{
				Naked: true,
				Value: w,
			})
		} else {
			p.followErr(p.pos, ds.Variant.Value, "names or assignments")
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

func (p *Parser) timeClause() *TimeClause {
	tc := &TimeClause{Time: p.pos}
	p.next()
	if !p.newLine {
		tc.Stmt = p.gotStmtPipe(p.stmt(p.pos))
	}
	return tc
}

func (p *Parser) coprocClause() *CoprocClause {
	cc := &CoprocClause{Coproc: p.pos}
	if p.next(); isBashCompoundCommand(p.tok, p.val) {
		// has no name
		cc.Stmt = p.gotStmtPipe(p.stmt(p.pos))
		return cc
	}
	if p.newLine {
		p.posErr(cc.Coproc, "coproc clause requires a command")
	}
	cc.Name = p.getLit()
	cc.Stmt = p.gotStmtPipe(p.stmt(p.pos))
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

func (p *Parser) letClause() *LetClause {
	lc := &LetClause{Let: p.pos}
	old := p.preNested(arithmExprLet)
	p.next()
	for !p.newLine && !stopToken(p.tok) && !p.peekRedir() {
		x := p.arithmExpr(0, true, false)
		if x == nil {
			break
		}
		lc.Exprs = append(lc.Exprs, x)
	}
	if len(lc.Exprs) == 0 {
		p.followErrExp(lc.Let, "let")
	}
	p.postNested(old)
	if p.tok == illegalTok {
		p.next()
	}
	return lc
}

func (p *Parser) bashFuncDecl() *FuncDecl {
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

func (p *Parser) callExpr(s *Stmt, w *Word, assign bool) Command {
	ce := p.call(w)
	if w == nil {
		ce.Args = ce.Args[:0]
	}
	if assign {
		ce.Assigns = append(ce.Assigns, p.getAssign(true))
	}
loop:
	for !p.newLine {
		switch p.tok {
		case _EOF, semicolon, and, or, andAnd, orOr, orAnd,
			dblSemicolon, semiAnd, dblSemiAnd, semiOr:
			break loop
		case _LitWord:
			if len(ce.Args) == 0 && p.hasValidIdent() {
				ce.Assigns = append(ce.Assigns, p.getAssign(true))
				break
			}
			ce.Args = append(ce.Args, p.word(
				p.wps(p.lit(p.pos, p.val)),
			))
			p.next()
		case _Lit:
			if len(ce.Args) == 0 && p.hasValidIdent() {
				ce.Assigns = append(ce.Assigns, p.getAssign(true))
				break
			}
			ce.Args = append(ce.Args, p.word(p.wordParts()))
		case bckQuote:
			if p.quote == subCmdBckquo {
				break loop
			}
			fallthrough
		case dollBrace, dollDblParen, dollParen, dollar, cmdIn, cmdOut,
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
				break loop
			}
			fallthrough
		default:
			p.curErr("a command can only contain words and redirects")
		}
	}
	if len(ce.Assigns) == 0 && len(ce.Args) == 0 {
		return nil
	}
	if len(ce.Args) == 0 {
		ce.Args = nil
	}
	return ce
}

func (p *Parser) funcDecl(name *Lit, pos Pos) *FuncDecl {
	fd := &FuncDecl{
		Position: pos,
		RsrvWord: pos != name.ValuePos,
		Name:     name,
	}
	if fd.Body, _ = p.getStmt(false, false); fd.Body == nil {
		p.followErr(fd.Pos(), "foo()", "a statement")
	}
	return fd
}
