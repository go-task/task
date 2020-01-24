// Copyright (c) 2016, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package syntax

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ParserOption is a function which can be passed to NewParser
// to alter its behaviour. To apply option to existing Parser
// call it directly, for example KeepComments(true)(parser).
type ParserOption func(*Parser)

// KeepComments makes the parser parse comments and attach them to
// nodes, as opposed to discarding them.
func KeepComments(enabled bool) ParserOption {
	return func(p *Parser) { p.keepComments = enabled }
}

type LangVariant int

const (
	LangBash LangVariant = iota
	LangPOSIX
	LangMirBSDKorn
)

// Variant changes the shell language variant that the parser will
// accept.
func Variant(l LangVariant) ParserOption {
	return func(p *Parser) { p.lang = l }
}

func (l LangVariant) String() string {
	switch l {
	case LangBash:
		return "bash"
	case LangPOSIX:
		return "posix"
	case LangMirBSDKorn:
		return "mksh"
	}
	return "unknown shell language variant"
}

// StopAt configures the lexer to stop at an arbitrary word, treating it
// as if it were the end of the input. It can contain any characters
// except whitespace, and cannot be over four bytes in size.
//
// This can be useful to embed shell code within another language, as
// one can use a special word to mark the delimiters between the two.
//
// As a word, it will only apply when following whitespace or a
// separating token. For example, StopAt("$$") will act on the inputs
// "foo $$" and "foo;$$", but not on "foo '$$'".
//
// The match is done by prefix, so the example above will also act on
// "foo $$bar".
func StopAt(word string) ParserOption {
	if len(word) > 4 {
		panic("stop word can't be over four bytes in size")
	}
	if strings.ContainsAny(word, " \t\n\r") {
		panic("stop word can't contain whitespace characters")
	}
	return func(p *Parser) { p.stopAt = []byte(word) }
}

// NewParser allocates a new Parser and applies any number of options.
func NewParser(options ...ParserOption) *Parser {
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
	p.f.Stmts, p.f.Last = p.stmtList()
	if p.err == nil {
		// EOF immediately after heredoc word so no newline to
		// trigger it
		p.doHeredocs()
	}
	return p.f, p.err
}

// Stmts reads and parses statements one at a time, calling a function
// each time one is parsed. If the function returns false, parsing is
// stopped and the function is not called again.
func (p *Parser) Stmts(r io.Reader, fn func(*Stmt) bool) error {
	p.reset()
	p.f = &File{}
	p.src = r
	p.rune()
	p.next()
	p.stmts(fn)
	if p.err == nil {
		// EOF immediately after heredoc word so no newline to
		// trigger it
		p.doHeredocs()
	}
	return p.err
}

type wrappedReader struct {
	*Parser
	io.Reader

	lastLine    uint16
	accumulated []*Stmt
	fn          func([]*Stmt) bool
}

func (w *wrappedReader) Read(p []byte) (n int, err error) {
	// If we lexed a newline for the first time, we just finished a line, so
	// we may need to give a callback for the edge cases below not covered
	// by Parser.Stmts.
	if (w.r == '\n' || w.r == escNewl) && w.npos.line > w.lastLine {
		if w.Incomplete() {
			// Incomplete statement; call back to print "> ".
			if !w.fn(w.accumulated) {
				return 0, io.EOF
			}
		} else if len(w.accumulated) == 0 {
			// Nothing was parsed; call back to print another "$ ".
			if !w.fn(nil) {
				return 0, io.EOF
			}
		}
		w.lastLine = w.npos.line
	}
	return w.Reader.Read(p)
}

// Interactive implements what is necessary to parse statements in an
// interactive shell. The parser will call the given function under two
// circumstances outlined below.
//
// If a line containing any number of statements is parsed, the function will be
// called with said statements.
//
// If a line ending in an incomplete statement is parsed, the function will be
// called with any fully parsed statents, and Parser.Incomplete will return
// true.
//
// One can imagine a simple interactive shell implementation as follows:
//
//         fmt.Fprintf(os.Stdout, "$ ")
//         parser.Interactive(os.Stdin, func(stmts []*syntax.Stmt) bool {
//                 if parser.Incomplete() {
//                         fmt.Fprintf(os.Stdout, "> ")
//                         return true
//                 }
//                 run(stmts)
//                 fmt.Fprintf(os.Stdout, "$ ")
//                 return true
//         }
//
// If the callback function returns false, parsing is stopped and the function
// is not called again.
func (p *Parser) Interactive(r io.Reader, fn func([]*Stmt) bool) error {
	w := wrappedReader{Parser: p, Reader: r, fn: fn}
	return p.Stmts(&w, func(stmt *Stmt) bool {
		w.accumulated = append(w.accumulated, stmt)
		// We finished parsing a statement and we're at a newline token,
		// so we finished fully parsing a number of statements. Call
		// back to run the statements and print "$ ".
		if p.tok == _Newl {
			if !fn(w.accumulated) {
				return false
			}
			w.accumulated = w.accumulated[:0]
			// The callback above would already print "$ ", so we
			// don't want the subsequent wrappedReader.Read to cause
			// another "$ " print thinking that nothing was parsed.
			w.lastLine = w.npos.line + 1
		}
		return true
	})
}

// Words reads and parses words one at a time, calling a function each time one
// is parsed. If the function returns false, parsing is stopped and the function
// is not called again.
//
// Newlines are skipped, meaning that multi-line input will work fine. If the
// parser encounters a token that isn't a word, such as a semicolon, an error
// will be returned.
//
// Note that the lexer doesn't currently tokenize spaces, so it may need to read
// a non-space byte such as a newline or a letter before finishing the parsing
// of a word. This will be fixed in the future.
func (p *Parser) Words(r io.Reader, fn func(*Word) bool) error {
	p.reset()
	p.f = &File{}
	p.src = r
	p.rune()
	p.next()
	for {
		p.got(_Newl)
		w := p.getWord()
		if w == nil {
			if p.tok != _EOF {
				p.curErr("%s is not a valid word", p.tok)
			}
			return p.err
		}
		if !fn(w) {
			return nil
		}
	}
}

// Document parses a single here-document word. That is, it parses the input as
// if they were lines following a <<EOF redirection.
//
// In practice, this is the same as parsing the input as if it were within
// double quotes, but without having to escape all double quote characters.
// Similarly, the here-document word parsed here cannot be ended by any
// delimiter other than reaching the end of the input.
func (p *Parser) Document(r io.Reader) (*Word, error) {
	p.reset()
	p.f = &File{}
	p.src = r
	p.rune()
	p.quote = hdocBody
	p.hdocStop = []byte("MVDAN_CC_SH_SYNTAX_EOF")
	p.parsingDoc = true
	p.next()
	w := p.getWord()
	return w, p.err
}

// Arithmetic parses a single arithmetic expression. That is, as if the input
// were within the $(( and )) tokens.
func (p *Parser) Arithmetic(r io.Reader) (ArithmExpr, error) {
	p.reset()
	p.f = &File{}
	p.src = r
	p.rune()
	p.quote = arithmExpr
	p.next()
	expr := p.arithmExpr(0, false, false)
	return expr, p.err
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

	spaced bool // whether tok has whitespace on its left

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

	stopAt []byte

	forbidNested bool

	// list of pending heredoc bodies
	buriedHdocs int
	heredocs    []*Redirect
	hdocStop    []byte
	parsingDoc  bool

	// openStmts is how many entire statements we're currently parsing. A
	// non-zero number means that we require certain tokens or words before
	// reaching EOF.
	openStmts int
	// openBquotes is how many levels of backquotes are open at the moment.
	openBquotes int

	// lastBquoteEsc is how many times the last backquote token was escaped
	lastBquoteEsc int
	// buriedBquotes is like openBquotes, but saved for when the parser
	// comes out of single quotes
	buriedBquotes int

	rxOpenParens int
	rxFirstPart  bool

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

// Incomplete reports whether the parser is waiting to read more bytes because
// it needs to finish properly parsing a statement.
//
// It is only safe to call while the parser is blocked on a read. For an example
// use case, see the documentation for Parser.Interactive.
func (p *Parser) Incomplete() bool {
	// If we're in a quote state other than noState, we're parsing a node
	// such as a double-quoted string.
	// If there are any open statements, we need to finish them.
	// If we're constructing a literal, we need to finish it.
	return p.quote != noState || p.openStmts > 0 || p.litBs != nil
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
	p.openStmts = 0
	p.heredocs, p.buriedHdocs = p.heredocs[:0], 0
	p.parsingDoc = false
	p.openBquotes, p.buriedBquotes = 0, 0
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

//go:generate stringer -type=quoteState

type quoteState uint32

const (
	noState quoteState = 1 << iota
	subCmd
	subCmdBckquo
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
	paramExpName
	paramExpSlice
	paramExpRepl
	paramExpExp
	arrayElems

	allKeepSpaces = paramExpRepl | dblQuotes | hdocBody |
		hdocBodyTabs | paramExpExp
	allRegTokens = noState | subCmd | subCmdBckquo | hdocWord |
		switchCase | arrayElems
	allArithmExpr = arithmExpr | arithmExprLet | arithmExprCmd |
		arithmExprBrack | paramExpSlice
	allParamReg = paramExpName | paramExpSlice
	allParamExp = allParamReg | paramExpRepl | paramExpExp | arithmExprBrack
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
	hdocs := p.heredocs[p.buriedHdocs:]
	if len(hdocs) == 0 {
		// Nothing do do; don't even issue a read.
		return
	}
	p.rune() // consume '\n', since we know p.tok == _Newl
	old := p.quote
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
			r.Hdoc = p.quotedHdocWord()
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

func (p *Parser) gotRsrv(val string) (Pos, bool) {
	pos := p.pos
	if p.tok == _LitWord && p.val == val {
		p.next()
		return pos, true
	}
	return pos, false
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
	pos, ok := p.gotRsrv(val)
	if !ok {
		p.followErr(lpos, left, fmt.Sprintf("%q", val))
	}
	return pos
}

func (p *Parser) followStmts(left string, lpos Pos, stops ...string) ([]*Stmt, []Comment) {
	if p.got(semicolon) {
		return nil, nil
	}
	newLine := p.got(_Newl)
	stmts, last := p.stmtList(stops...)
	if len(stmts) < 1 && !newLine {
		p.followErr(lpos, left, "a statement list")
	}
	return stmts, last
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
	pos, ok := p.gotRsrv(end)
	if !ok {
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
		p.w = 1
		p.tok = _EOF
	}
}

// IsIncomplete reports whether a Parser error could have been avoided with
// extra input bytes. For example, if an io.EOF was encountered while there was
// an unclosed quote or parenthesis.
func IsIncomplete(err error) bool {
	perr, ok := err.(ParseError)
	return ok && perr.Incomplete
}

// ParseError represents an error found when parsing a source file, from which
// the parser cannot recover.
type ParseError struct {
	Filename string
	Pos
	Text string

	Incomplete bool
}

func (e ParseError) Error() string {
	if e.Filename == "" {
		return fmt.Sprintf("%s: %s", e.Pos.String(), e.Text)
	}
	return fmt.Sprintf("%s:%s: %s", e.Filename, e.Pos.String(), e.Text)
}

// LangError is returned when the parser encounters code that is only valid in
// other shell language variants. The error includes what feature is not present
// in the current language variant, and what languages support it.
type LangError struct {
	Filename string
	Pos
	Feature string
	Langs   []LangVariant
}

func (e LangError) Error() string {
	var buf bytes.Buffer
	if e.Filename != "" {
		buf.WriteString(e.Filename + ":")
	}
	buf.WriteString(e.Pos.String() + ": ")
	buf.WriteString(e.Feature)
	if strings.HasSuffix(e.Feature, "s") {
		buf.WriteString(" are a ")
	} else {
		buf.WriteString(" is a ")
	}
	for i, lang := range e.Langs {
		if i > 0 {
			buf.WriteString("/")
		}
		buf.WriteString(lang.String())
	}
	buf.WriteString(" feature")
	return buf.String()
}

func (p *Parser) posErr(pos Pos, format string, a ...interface{}) {
	p.errPass(ParseError{
		Filename:   p.f.Name,
		Pos:        pos,
		Text:       fmt.Sprintf(format, a...),
		Incomplete: p.tok == _EOF && p.Incomplete(),
	})
}

func (p *Parser) curErr(format string, a ...interface{}) {
	p.posErr(p.pos, format, a...)
}

func (p *Parser) langErr(pos Pos, feature string, langs ...LangVariant) {
	p.errPass(LangError{
		Filename: p.f.Name,
		Pos:      pos,
		Feature:  feature,
		Langs:    langs,
	})
}

func (p *Parser) stmts(fn func(*Stmt) bool, stops ...string) {
	gotEnd := true
loop:
	for p.tok != _EOF {
		newLine := p.got(_Newl)
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
			if p.backquoteEnd() {
				break loop
			}
		case dblSemicolon, semiAnd, dblSemiAnd, semiOr:
			if p.quote == switchCase {
				break loop
			}
			p.curErr("%s can only be used in a case clause", p.tok)
		}
		if !newLine && !gotEnd {
			p.curErr("statements must be separated by &, ; or a newline")
		}
		if p.tok == _EOF {
			break
		}
		p.openStmts++
		s := p.getStmt(true, false, false)
		p.openStmts--
		if s == nil {
			p.invalidStmtStart()
			break
		}
		gotEnd = s.Semicolon.IsValid()
		if !fn(s) {
			break
		}
	}
}

func (p *Parser) stmtList(stops ...string) ([]*Stmt, []Comment) {
	var stmts []*Stmt
	var last []Comment
	fn := func(s *Stmt) bool {
		if stmts == nil {
			stmts = p.stList()
		}
		stmts = append(stmts, s)
		return true
	}
	p.stmts(fn, stops...)
	split := len(p.accComs)
	if p.tok == _LitWord && (p.val == "elif" || p.val == "else" || p.val == "fi") {
		// Split the comments, so that any aligned with an opening token
		// get attached to it. For example:
		//
		//     if foo; then
		//         # inside the body
		//     # document the else
		//     else
		//     fi
		// TODO(mvdan): look into deduplicating this with similar logic
		// in caseItems.
		for i := len(p.accComs) - 1; i >= 0; i-- {
			c := p.accComs[i]
			if c.Pos().Col() != p.pos.Col() {
				break
			}
			split = i
		}
	}
	last = p.accComs[:split]
	p.accComs = p.accComs[split:]
	return stmts, last
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
	if parts := p.wordParts(); len(parts) > 0 && p.err == nil {
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
			cs.Stmts, cs.Last = p.stmtList("}")
			p.postNested(old)
			pos, ok := p.gotRsrv("}")
			if !ok {
				p.matchingErr(cs.Left, "${", "}")
			}
			cs.Right = pos
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
				p.langErr(ar.Pos(), "unsigned expressions", LangMirBSDKorn)
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
		cs.Stmts, cs.Last = p.stmtList()
		p.postNested(old)
		cs.Right = p.matched(cs.Left, leftParen, rightParen)
		return cs
	case dollar:
		r := p.r
		switch {
		case singleRuneParam(r):
			p.tok, p.val = _LitWord, string(r)
			p.rune()
		case 'a' <= r && r <= 'z', 'A' <= r && r <= 'Z',
			'0' <= r && r <= '9', r == '_', r == '\\':
			p.advanceNameCont(r)
		default:
			l := p.lit(p.pos, "$")
			p.next()
			return l
		}
		p.ensureNoNested()
		pe := &ParamExp{Dollar: p.pos, Short: true}
		p.pos = posAddCol(p.pos, 1)
		pe.Param = p.getLit()
		if pe.Param != nil && pe.Param.Value == "" {
			l := p.lit(pe.Dollar, "$")
			// e.g. "$\\\"" within double quotes, so we must
			// keep the rest of the literal characters.
			l.ValueEnd = posAddCol(l.ValuePos, 1)
			return l
		}
		return pe
	case cmdIn, cmdOut:
		p.ensureNoNested()
		ps := &ProcSubst{Op: ProcOperator(p.tok), OpPos: p.pos}
		old := p.preNested(subCmd)
		p.next()
		ps.Stmts, ps.Last = p.stmtList()
		p.postNested(old)
		ps.Rparen = p.matched(ps.OpPos, token(ps.Op), rightParen)
		return ps
	case sglQuote, dollSglQuote:
		sq := &SglQuoted{Left: p.pos, Dollar: p.tok == dollSglQuote}
		r := p.r
		for p.newLit(r); ; r = p.rune() {
			switch r {
			case '\\':
				if sq.Dollar {
					p.rune()
				}
			case '\'':
				sq.Right = p.getPos()
				sq.Value = p.endLit()

				// restore openBquotes
				p.openBquotes = p.buriedBquotes
				p.buriedBquotes = 0

				p.rune()
				p.next()
				return sq
			case escNewl:
				p.litBs = append(p.litBs, '\\', '\n')
			case utf8.RuneSelf:
				p.tok = _EOF
				p.quoteErr(sq.Pos(), sglQuote)
				return nil
			}
		}
	case dblQuote, dollDblQuote:
		if p.quote == dblQuotes {
			// p.tok == dblQuote, as "foo$" puts $ in the lit
			return nil
		}
		return p.dblQuoted()
	case bckQuote:
		if p.backquoteEnd() {
			return nil
		}
		p.ensureNoNested()
		cs := &CmdSubst{Left: p.pos, Backquotes: true}
		old := p.preNested(subCmdBckquo)
		p.openBquotes++

		// The lexer didn't call p.rune for us, so that it could have
		// the right p.openBquotes to properly handle backslashes.
		p.rune()

		p.next()
		cs.Stmts, cs.Last = p.stmtList()
		if p.tok == bckQuote && p.lastBquoteEsc < p.openBquotes-1 {
			// e.g. found ` before the nested backquote \` was closed.
			p.tok = _EOF
			p.quoteErr(cs.Pos(), bckQuote)
		}
		p.postNested(old)
		p.openBquotes--
		cs.Right = p.pos

		// Like above, the lexer didn't call p.rune for us.
		p.rune()
		if !p.got(bckQuote) {
			p.quoteErr(cs.Pos(), bckQuote)
		}
		return cs
	case globQuest, globStar, globPlus, globAt, globExcl:
		if p.lang == LangPOSIX {
			p.langErr(p.pos, "extended globs", LangBash, LangMirBSDKorn)
		}
		eg := &ExtGlob{Op: GlobOperator(p.tok), OpPos: p.pos}
		lparens := 1
		r := p.r
	globLoop:
		for p.newLit(r); ; r = p.rune() {
			switch r {
			case utf8.RuneSelf:
				break globLoop
			case '(':
				lparens++
			case ')':
				if lparens--; lparens == 0 {
					break globLoop
				}
			}
		}
		eg.Pattern = p.lit(posAddCol(eg.OpPos, 2), p.endLit())
		p.rune()
		p.next()
		if lparens != 0 {
			p.matchingErr(eg.OpPos, eg.Op, rightParen)
		}
		return eg
	default:
		return nil
	}
}

func (p *Parser) dblQuoted() *DblQuoted {
	q := &DblQuoted{Left: p.pos, Dollar: p.tok == dollDblQuote}
	old := p.quote
	p.quote = dblQuotes
	p.next()
	q.Parts = p.wordParts()
	p.quote = old
	q.Right = p.pos
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
	case TernQuest, TernColon:
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
	p.got(_Newl)
	newLevel := arithmOpLevel(BinAritOperator(p.tok))
	if !tern && p.tok == colon && p.quote == paramExpSlice {
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
	case TernColon:
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
	b.Y = p.arithmExpr(newLevel, compact, b.Op == TernQuest)
	if b.Y == nil {
		p.followErrExp(b.OpPos, b.Op.String())
	}
	if b.Op == TernQuest {
		if b2, ok := b.Y.(*BinaryArithm); !ok || b2.Op != TernColon {
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
	p.got(_Newl)
	var x ArithmExpr
	switch p.tok {
	case exclMark, tilde:
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
		pe := &ParamExp{Dollar: l.ValuePos, Short: true, Param: l}
		pe.Index = p.eitherIndex()
		x = p.word(p.wps(pe))
	case bckQuote:
		if p.quote == arithmExprLet && p.openBquotes > 0 {
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

func singleRuneParam(r rune) bool {
	switch r {
	case '@', '*', '#', '$', '?', '!', '-',
		'0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return true
	}
	return false
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
			if p.lang == LangPOSIX {
				p.langErr(p.pos, "${!foo}", LangBash, LangMirBSDKorn)
			}
			pe.Excl = true
			p.next()
		}
	}
	op := p.tok
	switch p.tok {
	case _Lit, _LitWord:
		if !numberLiteral(p.val) && !ValidName(p.val) {
			p.curErr("invalid parameter name")
		}
		pe.Param = p.lit(p.pos, p.val)
		p.next()
	case quest, minus:
		if pe.Length && p.r != '}' {
			// actually ${#-default}, not ${#-}; fix the ambiguity
			pe.Length = false
			pe.Param = p.lit(posAddCol(p.pos, -1), "#")
			pe.Param.ValueEnd = p.pos
			break
		}
		fallthrough
	case at, star, hash, exclMark, dollar:
		pe.Param = p.lit(p.pos, p.tok.String())
		p.next()
	default:
		p.curErr("parameter expansion requires a literal")
	}
	switch p.tok {
	case _Lit, _LitWord:
		p.curErr("%s cannot be followed by a word", op)
	case rightBrace:
		pe.Rbrace = p.pos
		p.quote = old
		p.next()
		return pe
	case leftBrack:
		if p.lang == LangPOSIX {
			p.langErr(p.pos, "arrays", LangBash, LangMirBSDKorn)
		}
		if !ValidName(pe.Param.Value) {
			p.curErr("cannot index a special parameter name")
		}
		pe.Index = p.eitherIndex()
	}
	if p.tok == rightBrace {
		pe.Rbrace = p.pos
		p.quote = old
		p.next()
		return pe
	}
	if p.tok != _EOF && (pe.Length || pe.Width) {
		p.curErr("cannot combine multiple parameter expansion operators")
	}
	switch p.tok {
	case slash, dblSlash:
		// pattern search and replace
		if p.lang == LangPOSIX {
			p.langErr(p.pos, "search and replace", LangBash, LangMirBSDKorn)
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
		// slicing
		if p.lang == LangPOSIX {
			p.langErr(p.pos, "slicing", LangBash, LangMirBSDKorn)
		}
		pe.Slice = &Slice{}
		colonPos := p.pos
		p.quote = paramExpSlice
		if p.next(); p.tok != colon {
			pe.Slice.Offset = p.followArithm(colon, colonPos)
		}
		colonPos = p.pos
		if p.got(colon) {
			pe.Slice.Length = p.followArithm(colon, colonPos)
		}
	case caret, dblCaret, comma, dblComma:
		// upper/lower case
		if p.lang != LangBash {
			p.langErr(p.pos, "this expansion operator", LangBash)
		}
		pe.Exp = p.paramExpExp()
	case at, star:
		switch {
		case p.tok == at && p.lang == LangPOSIX:
			p.langErr(p.pos, "this expansion operator", LangBash, LangMirBSDKorn)
		case p.tok == star && !pe.Excl:
			p.curErr("not a valid parameter expansion operator: %v", p.tok)
		case pe.Excl:
			pe.Names = ParNamesOperator(p.tok)
			p.next()
		default:
			pe.Exp = p.paramExpExp()
		}
	case plus, colPlus, minus, colMinus, quest, colQuest, assgn, colAssgn,
		perc, dblPerc, hash, dblHash:
		pe.Exp = p.paramExpExp()
	case _EOF:
	default:
		p.curErr("not a valid parameter expansion operator: %v", p.tok)
	}
	p.quote = old
	pe.Rbrace = p.pos
	p.matched(pe.Dollar, dollBrace, rightBrace)
	return pe
}

func (p *Parser) paramExpExp() *Expansion {
	op := ParExpOperator(p.tok)
	p.quote = paramExpExp
	p.next()
	if op == OtherParamOps {
		switch p.tok {
		case _Lit, _LitWord:
		default:
			p.curErr("@ expansion operator requires a literal")
		}
		switch p.val {
		case "Q", "E", "P", "A", "a":
		default:
			p.curErr("invalid @ expansion operator")
		}
	}
	return &Expansion{Op: op, Word: p.getWord()}
}

func (p *Parser) eitherIndex() ArithmExpr {
	old := p.quote
	lpos := p.pos
	p.quote = arithmExprBrack
	p.next()
	if p.tok == star || p.tok == at {
		p.tok, p.val = _LitWord, p.tok.String()
	}
	expr := p.followArithm(leftBrack, lpos)
	p.quote = old
	p.matched(lpos, leftBrack, rightBrack)
	return expr
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
	case _EOF, _Newl, semicolon, and, or, andAnd, orOr, orAnd, dblSemicolon,
		semiAnd, dblSemiAnd, semiOr, rightParen:
		return true
	}
	return false
}

func (p *Parser) backquoteEnd() bool {
	return p.quote == subCmdBckquo && p.lastBquoteEsc < p.openBquotes
}

// ValidName returns whether val is a valid name as per the POSIX spec.
func ValidName(val string) bool {
	if val == "" {
		return false
	}
	for i, r := range val {
		switch {
		case 'a' <= r && r <= 'z':
		case 'A' <= r && r <= 'Z':
		case r == '_':
		case i > 0 && '0' <= r && r <= '9':
		default:
			return false
		}
	}
	return true
}

func numberLiteral(val string) bool {
	for _, r := range val {
		if '0' > r || r > '9' {
			return false
		}
	}
	return true
}

func (p *Parser) hasValidIdent() bool {
	if p.tok != _Lit && p.tok != _LitWord {
		return false
	}
	if end := p.eqlOffs; end > 0 {
		if p.val[end-1] == '+' && p.lang != LangPOSIX {
			end--
		}
		if ValidName(p.val[:end]) {
			return true
		}
	}
	return p.r == '['
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
		p.pos = posAddCol(p.pos, 1)
		as.Index = p.eitherIndex()
		if p.spaced || stopToken(p.tok) {
			if needEqual {
				p.followErr(as.Pos(), "a[b]", "=")
			} else {
				as.Naked = true
				return as
			}
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
			p.langErr(p.pos, "arrays", LangBash, LangMirBSDKorn)
		}
		if as.Index != nil {
			p.curErr("arrays cannot be nested")
		}
		as.Array = &ArrayExpr{Lparen: p.pos}
		newQuote := p.quote
		if p.lang == LangBash {
			newQuote = arrayElems
		}
		old := p.preNested(newQuote)
		p.next()
		p.got(_Newl)
		for p.tok != _EOF && p.tok != rightParen {
			ae := &ArrayElem{}
			ae.Comments, p.accComs = p.accComs, nil
			if p.tok == leftBrack {
				left := p.pos
				ae.Index = p.eitherIndex()
				p.follow(left, `"[x]"`, assgn)
			}
			if ae.Value = p.getWord(); ae.Value == nil {
				switch p.tok {
				case leftParen:
					p.curErr("arrays cannot be nested")
					return nil
				case _Newl, rightParen, leftBrack:
					// TODO: support [index]=[
				default:
					p.curErr("array element values must be words")
					return nil
				}
			}
			if len(p.accComs) > 0 {
				c := p.accComs[0]
				if c.Pos().Line() == ae.End().Line() {
					ae.Comments = append(ae.Comments, c)
					p.accComs = p.accComs[1:]
				}
			}
			as.Array.Elems = append(as.Array.Elems, ae)
			p.got(_Newl)
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
	var r *Redirect
	if s.Redirs == nil {
		var alloc struct {
			redirs [4]*Redirect
			redir  Redirect
		}
		s.Redirs = alloc.redirs[:0]
		r = &alloc.redir
		s.Redirs = append(s.Redirs, r)
	} else {
		r = &Redirect{}
		s.Redirs = append(s.Redirs, r)
	}
	r.N = p.getLit()
	if p.lang != LangBash && r.N != nil && r.N.Value[0] == '{' {
		p.langErr(r.N.Pos(), "{varname} redirects", LangBash)
	}
	r.Op, r.OpPos = RedirOperator(p.tok), p.pos
	p.next()
	switch r.Op {
	case Hdoc, DashHdoc:
		old := p.quote
		p.quote, p.forbidNested = hdocWord, true
		p.heredocs = append(p.heredocs, r)
		r.Word = p.followWordTok(token(r.Op), r.OpPos)
		p.quote, p.forbidNested = old, false
		if p.tok == _Newl {
			if len(p.accComs) > 0 {
				c := p.accComs[0]
				if c.Pos().Line() == s.End().Line() {
					s.Comments = append(s.Comments, c)
					p.accComs = p.accComs[1:]
				}
			}
			p.doHeredocs()
		}
	default:
		r.Word = p.followWordTok(token(r.Op), r.OpPos)
	}
}

func (p *Parser) getStmt(readEnd, binCmd, fnBody bool) *Stmt {
	pos, ok := p.gotRsrv("!")
	s := p.stmt(pos)
	if ok {
		s.Negated = true
		if stopToken(p.tok) {
			p.posErr(s.Pos(), `"!" cannot form a statement alone`)
		}
		if _, ok := p.gotRsrv("!"); ok {
			p.posErr(s.Pos(), `cannot negate a command multiple times`)
		}
	}
	if s = p.gotStmtPipe(s, false); s == nil || p.err != nil {
		return nil
	}
	// instead of using recursion, iterate manually
	for p.tok == andAnd || p.tok == orOr {
		if binCmd {
			// left associativity: in a list of BinaryCmds, the
			// right recursion should only read a single element
			return s
		}
		b := &BinaryCmd{
			OpPos: p.pos,
			Op:    BinCmdOperator(p.tok),
			X:     s,
		}
		p.next()
		p.got(_Newl)
		b.Y = p.getStmt(false, true, false)
		if b.Y == nil || p.err != nil {
			p.followErr(b.OpPos, b.Op.String(), "a statement")
			return nil
		}
		s = p.stmt(s.Position)
		s.Cmd = b
		s.Comments, b.X.Comments = b.X.Comments, nil
	}
	if readEnd {
		switch p.tok {
		case semicolon:
			s.Semicolon = p.pos
			p.next()
		case and:
			s.Semicolon = p.pos
			p.next()
			s.Background = true
		case orAnd:
			s.Semicolon = p.pos
			p.next()
			s.Coprocess = true
		}
	}
	if len(p.accComs) > 0 && !binCmd && !fnBody {
		c := p.accComs[0]
		if c.Pos().Line() == s.End().Line() {
			s.Comments = append(s.Comments, c)
			p.accComs = p.accComs[1:]
		}
	}
	return s
}

func (p *Parser) gotStmtPipe(s *Stmt, binCmd bool) *Stmt {
	s.Comments, p.accComs = p.accComs, nil
	switch p.tok {
	case _LitWord:
		switch p.val {
		case "{":
			p.block(s)
		case "if":
			p.ifClause(s)
		case "while", "until":
			p.whileClause(s, p.val == "until")
		case "for":
			p.forClause(s)
		case "case":
			p.caseClause(s)
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
				p.testClause(s)
			}
		case "]]":
			if p.lang != LangPOSIX {
				p.curErr(`%q can only be used to close a test`,
					p.val)
			}
		case "let":
			if p.lang != LangPOSIX {
				p.letClause(s)
			}
		case "function":
			if p.lang != LangPOSIX {
				p.bashFuncDecl(s)
			}
		case "declare":
			if p.lang == LangBash {
				p.declClause(s)
			}
		case "local", "export", "readonly", "typeset", "nameref":
			if p.lang != LangPOSIX {
				p.declClause(s)
			}
		case "time":
			if p.lang != LangPOSIX {
				p.timeClause(s)
			}
		case "coproc":
			if p.lang == LangBash {
				p.coprocClause(s)
			}
		case "select":
			if p.lang != LangPOSIX {
				p.selectClause(s)
			}
		}
		if s.Cmd != nil {
			break
		}
		if p.hasValidIdent() {
			p.callExpr(s, nil, true)
			break
		}
		name := p.lit(p.pos, p.val)
		if p.next(); p.got(leftParen) {
			p.follow(name.ValuePos, "foo(", rightParen)
			if p.lang == LangPOSIX && !ValidName(name.Value) {
				p.posErr(name.Pos(), "invalid func name")
			}
			p.funcDecl(s, name, name.ValuePos)
		} else {
			p.callExpr(s, p.word(p.wps(name)), false)
		}
	case rdrOut, appOut, rdrIn, dplIn, dplOut, clbOut, rdrInOut,
		hdoc, dashHdoc, wordHdoc, rdrAll, appAll, _LitRedir:
		p.doRedirect(s)
		p.callExpr(s, nil, false)
	case bckQuote:
		if p.backquoteEnd() {
			return nil
		}
		fallthrough
	case _Lit, dollBrace, dollDblParen, dollParen, dollar, cmdIn, cmdOut,
		sglQuote, dollSglQuote, dblQuote, dollDblQuote, dollBrack,
		globQuest, globStar, globPlus, globAt, globExcl:
		if p.hasValidIdent() {
			p.callExpr(s, nil, true)
			break
		}
		w := p.word(p.wordParts())
		if p.got(leftParen) && p.err == nil {
			p.posErr(w.Pos(), "invalid func name")
		}
		p.callExpr(s, w, false)
	case leftParen:
		p.subshell(s)
	case dblLeftParen:
		p.arithmExpCmd(s)
	default:
		if len(s.Redirs) == 0 {
			return nil
		}
	}
	for p.peekRedir() {
		p.doRedirect(s)
	}
	// instead of using recursion, iterate manually
	for p.tok == or || p.tok == orAnd {
		if binCmd {
			// left associativity: in a list of BinaryCmds, the
			// right recursion should only read a single element
			return s
		}
		if p.tok == orAnd && p.lang == LangMirBSDKorn {
			// No need to check for LangPOSIX, as on that language
			// we parse |& as two tokens.
			break
		}
		b := &BinaryCmd{OpPos: p.pos, Op: BinCmdOperator(p.tok), X: s}
		p.next()
		p.got(_Newl)
		if b.Y = p.gotStmtPipe(p.stmt(p.pos), true); b.Y == nil || p.err != nil {
			p.followErr(b.OpPos, b.Op.String(), "a statement")
			break
		}
		s = p.stmt(s.Position)
		s.Cmd = b
		s.Comments, b.X.Comments = b.X.Comments, nil
		// in "! x | y", the bang applies to the entire pipeline
		s.Negated = b.X.Negated
		b.X.Negated = false
	}
	return s
}

func (p *Parser) subshell(s *Stmt) {
	sub := &Subshell{Lparen: p.pos}
	old := p.preNested(subCmd)
	p.next()
	sub.Stmts, sub.Last = p.stmtList()
	p.postNested(old)
	sub.Rparen = p.matched(sub.Lparen, leftParen, rightParen)
	s.Cmd = sub
}

func (p *Parser) arithmExpCmd(s *Stmt) {
	ar := &ArithmCmd{Left: p.pos}
	old := p.preNested(arithmExprCmd)
	p.next()
	if p.got(hash) {
		if p.lang != LangMirBSDKorn {
			p.langErr(ar.Pos(), "unsigned expressions", LangMirBSDKorn)
		}
		ar.Unsigned = true
	}
	ar.X = p.followArithm(dblLeftParen, ar.Left)
	ar.Right = p.arithmEnd(dblLeftParen, ar.Left, old)
	s.Cmd = ar
}

func (p *Parser) block(s *Stmt) {
	b := &Block{Lbrace: p.pos}
	p.next()
	b.Stmts, b.Last = p.stmtList("}")
	pos, ok := p.gotRsrv("}")
	b.Rbrace = pos
	if !ok {
		p.matchingErr(b.Lbrace, "{", "}")
	}
	s.Cmd = b
}

func (p *Parser) ifClause(s *Stmt) {
	rootIf := &IfClause{Position: p.pos}
	p.next()
	rootIf.Cond, rootIf.CondLast = p.followStmts("if", rootIf.Position, "then")
	rootIf.ThenPos = p.followRsrv(rootIf.Position, "if <cond>", "then")
	rootIf.Then, rootIf.ThenLast = p.followStmts("then", rootIf.ThenPos, "fi", "elif", "else")
	curIf := rootIf
	for p.tok == _LitWord && p.val == "elif" {
		elf := &IfClause{Position: p.pos}
		curIf.Last = p.accComs
		p.accComs = nil
		p.next()
		elf.Cond, elf.CondLast = p.followStmts("elif", elf.Position, "then")
		elf.ThenPos = p.followRsrv(elf.Position, "elif <cond>", "then")
		elf.Then, elf.ThenLast = p.followStmts("then", elf.ThenPos, "fi", "elif", "else")
		curIf.Else = elf
		curIf = elf
	}
	if elsePos, ok := p.gotRsrv("else"); ok {
		curIf.Last = p.accComs
		p.accComs = nil
		els := &IfClause{Position: elsePos}
		els.Then, els.ThenLast = p.followStmts("else", els.Position, "fi")
		curIf.Else = els
		curIf = els
	}
	curIf.Last = p.accComs
	p.accComs = nil
	rootIf.FiPos = p.stmtEnd(rootIf, "if", "fi")
	for els := rootIf.Else; els != nil; els = els.Else {
		// All the nested IfClauses share the same FiPos.
		els.FiPos = rootIf.FiPos
	}
	s.Cmd = rootIf
}

func (p *Parser) whileClause(s *Stmt, until bool) {
	wc := &WhileClause{WhilePos: p.pos, Until: until}
	rsrv := "while"
	rsrvCond := "while <cond>"
	if wc.Until {
		rsrv = "until"
		rsrvCond = "until <cond>"
	}
	p.next()
	wc.Cond, wc.CondLast = p.followStmts(rsrv, wc.WhilePos, "do")
	wc.DoPos = p.followRsrv(wc.WhilePos, rsrvCond, "do")
	wc.Do, wc.DoLast = p.followStmts("do", wc.DoPos, "done")
	wc.DonePos = p.stmtEnd(wc, rsrv, "done")
	s.Cmd = wc
}

func (p *Parser) forClause(s *Stmt) {
	fc := &ForClause{ForPos: p.pos}
	p.next()
	fc.Loop = p.loop(fc.ForPos)
	fc.DoPos = p.followRsrv(fc.ForPos, "for foo [in words]", "do")

	s.Comments = append(s.Comments, p.accComs...)
	p.accComs = nil
	fc.Do, fc.DoLast = p.followStmts("do", fc.DoPos, "done")
	fc.DonePos = p.stmtEnd(fc, "for", "done")
	s.Cmd = fc
}

func (p *Parser) loop(fpos Pos) Loop {
	if p.lang != LangBash {
		switch p.tok {
		case leftParen, dblLeftParen:
			p.langErr(p.pos, "c-style fors", LangBash)
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
		p.got(semicolon)
		p.got(_Newl)
		return cl
	}
	return p.wordIter("for", fpos)
}

func (p *Parser) wordIter(ftok string, fpos Pos) *WordIter {
	wi := &WordIter{}
	if wi.Name = p.getLit(); wi.Name == nil {
		p.followErr(fpos, ftok, "a literal")
	}
	if p.got(semicolon) {
		p.got(_Newl)
		return wi
	}
	p.got(_Newl)
	if pos, ok := p.gotRsrv("in"); ok {
		wi.InPos = pos
		for !stopToken(p.tok) {
			if w := p.getWord(); w == nil {
				p.curErr("word list can only contain words")
			} else {
				wi.Items = append(wi.Items, w)
			}
		}
		p.got(semicolon)
		p.got(_Newl)
	} else if p.tok == _LitWord && p.val == "do" {
	} else {
		p.followErr(fpos, ftok+" foo", `"in", "do", ;, or a newline`)
	}
	return wi
}

func (p *Parser) selectClause(s *Stmt) {
	fc := &ForClause{ForPos: p.pos, Select: true}
	p.next()
	fc.Loop = p.wordIter("select", fc.ForPos)
	fc.DoPos = p.followRsrv(fc.ForPos, "select foo [in words]", "do")
	fc.Do, fc.DoLast = p.followStmts("do", fc.DoPos, "done")
	fc.DonePos = p.stmtEnd(fc, "select", "done")
	s.Cmd = fc
}

func (p *Parser) caseClause(s *Stmt) {
	cc := &CaseClause{Case: p.pos}
	p.next()
	cc.Word = p.followWord("case", cc.Case)
	end := "esac"
	p.got(_Newl)
	if _, ok := p.gotRsrv("{"); ok {
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
	s.Cmd = cc
}

func (p *Parser) caseItems(stop string) (items []*CaseItem) {
	p.got(_Newl)
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
		ci.Stmts, ci.Last = p.stmtList(stop)
		p.postNested(old)
		switch p.tok {
		case dblSemicolon, semiAnd, dblSemiAnd, semiOr:
		default:
			ci.Op = Break
			items = append(items, ci)
			return
		}
		ci.Last = append(ci.Last, p.accComs...)
		p.accComs = nil
		ci.OpPos = p.pos
		ci.Op = CaseOperator(p.tok)
		p.next()
		p.got(_Newl)
		split := len(p.accComs)
		if p.tok == _LitWord && p.val != stop {
			for i := len(p.accComs) - 1; i >= 0; i-- {
				c := p.accComs[i]
				if c.Pos().Col() != p.pos.Col() {
					break
				}
				split = i
			}
		}
		ci.Comments = append(ci.Comments, p.accComs[:split]...)
		p.accComs = p.accComs[split:]
		items = append(items, ci)
	}
	return
}

func (p *Parser) testClause(s *Stmt) {
	tc := &TestClause{Left: p.pos}
	p.next()
	if _, ok := p.gotRsrv("]]"); ok || p.tok == _EOF {
		p.posErr(tc.Left, "test clause requires at least one expression")
	}
	tc.X = p.testExpr(dblLeftBrack, tc.Left, false)
	tc.Right = p.pos
	if _, ok := p.gotRsrv("]]"); !ok {
		p.matchingErr(tc.Left, "[[", "]]")
	}
	s.Cmd = tc
}

func (p *Parser) testExpr(ftok token, fpos Pos, pastAndOr bool) TestExpr {
	p.got(_Newl)
	var left TestExpr
	if pastAndOr {
		left = p.testExprBase(ftok, fpos)
	} else {
		left = p.testExpr(ftok, fpos, true)
	}
	if left == nil {
		return left
	}
	p.got(_Newl)
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
	// Save the previous quoteState, since we change it in TsReMatch.
	oldQuote := p.quote

	switch b.Op {
	case AndTest, OrTest:
		p.next()
		if b.Y = p.testExpr(token(b.Op), b.OpPos, false); b.Y == nil {
			p.followErrExp(b.OpPos, b.Op.String())
		}
	case TsReMatch:
		if p.lang != LangBash {
			p.langErr(p.pos, "regex tests", LangBash)
		}
		p.rxOpenParens = 0
		p.rxFirstPart = true
		// TODO(mvdan): Using nested states within a regex will break in
		// all sorts of ways. The better fix is likely to use a stop
		// token, like we do with heredocs.
		p.quote = testRegexp
		fallthrough
	default:
		if _, ok := b.X.(*Word); !ok {
			p.posErr(b.OpPos, "expected %s, %s or %s after complex expr",
				AndTest, OrTest, "]]")
		}
		p.next()
		b.Y = p.followWordTok(token(b.Op), b.OpPos)
	}
	p.quote = oldQuote
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
		return p.followWordTok(ftok, fpos)
	}
}

func (p *Parser) declClause(s *Stmt) {
	ds := &DeclClause{Variant: p.lit(p.pos, p.val)}
	p.next()
	for !stopToken(p.tok) && !p.peekRedir() {
		if p.hasValidIdent() {
			ds.Args = append(ds.Args, p.getAssign(false))
		} else if p.eqlOffs > 0 {
			p.curErr("invalid var name")
		} else if p.tok == _LitWord && ValidName(p.val) {
			ds.Args = append(ds.Args, &Assign{
				Naked: true,
				Name:  p.getLit(),
			})
		} else if w := p.getWord(); w != nil {
			ds.Args = append(ds.Args, &Assign{
				Naked: true,
				Value: w,
			})
		} else {
			p.followErr(p.pos, ds.Variant.Value, "names or assignments")
		}
	}
	s.Cmd = ds
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

func (p *Parser) timeClause(s *Stmt) {
	tc := &TimeClause{Time: p.pos}
	p.next()
	if _, ok := p.gotRsrv("-p"); ok {
		tc.PosixFormat = true
	}
	tc.Stmt = p.gotStmtPipe(p.stmt(p.pos), false)
	s.Cmd = tc
}

func (p *Parser) coprocClause(s *Stmt) {
	cc := &CoprocClause{Coproc: p.pos}
	if p.next(); isBashCompoundCommand(p.tok, p.val) {
		// has no name
		cc.Stmt = p.gotStmtPipe(p.stmt(p.pos), false)
		s.Cmd = cc
		return
	}
	cc.Name = p.getWord()
	cc.Stmt = p.gotStmtPipe(p.stmt(p.pos), false)
	if cc.Stmt == nil {
		if cc.Name == nil {
			p.posErr(cc.Coproc, "coproc clause requires a command")
			return
		}
		// name was in fact the stmt
		cc.Stmt = p.stmt(cc.Name.Pos())
		cc.Stmt.Cmd = p.call(cc.Name)
		cc.Name = nil
	} else if cc.Name != nil {
		if call, ok := cc.Stmt.Cmd.(*CallExpr); ok {
			// name was in fact the start of a call
			call.Args = append([]*Word{cc.Name}, call.Args...)
			cc.Name = nil
		}
	}
	s.Cmd = cc
}

func (p *Parser) letClause(s *Stmt) {
	lc := &LetClause{Let: p.pos}
	old := p.preNested(arithmExprLet)
	p.next()
	for !stopToken(p.tok) && !p.peekRedir() {
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
	s.Cmd = lc
}

func (p *Parser) bashFuncDecl(s *Stmt) {
	fpos := p.pos
	if p.next(); p.tok != _LitWord {
		if w := p.followWord("function", fpos); p.err == nil {
			p.posErr(w.Pos(), "invalid func name")
		}
	}
	name := p.lit(p.pos, p.val)
	if p.next(); p.got(leftParen) {
		p.follow(name.ValuePos, "foo(", rightParen)
	}
	p.funcDecl(s, name, fpos)
}

func (p *Parser) callExpr(s *Stmt, w *Word, assign bool) {
	ce := p.call(w)
	if w == nil {
		ce.Args = ce.Args[:0]
	}
	if assign {
		ce.Assigns = append(ce.Assigns, p.getAssign(true))
	}
loop:
	for {
		switch p.tok {
		case _EOF, _Newl, semicolon, and, or, andAnd, orOr, orAnd,
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
			if p.backquoteEnd() {
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
		return
	}
	if len(ce.Args) == 0 {
		ce.Args = nil
	} else {
		for _, asgn := range ce.Assigns {
			if asgn.Index != nil || asgn.Array != nil {
				p.posErr(asgn.Pos(), "inline variables cannot be arrays")
			}
		}
	}
	s.Cmd = ce
}

func (p *Parser) funcDecl(s *Stmt, name *Lit, pos Pos) {
	fd := &FuncDecl{
		Position: pos,
		RsrvWord: pos != name.ValuePos,
		Name:     name,
	}
	p.got(_Newl)
	if fd.Body = p.getStmt(false, false, true); fd.Body == nil {
		p.followErr(fd.Pos(), "foo()", "a statement")
	}
	s.Cmd = fd
}
