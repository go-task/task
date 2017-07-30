// Copyright (c) 2016, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package syntax

import (
	"bytes"
	"io"
	"unicode/utf8"
)

// bytes that form or start a token
func regOps(r rune) bool {
	switch r {
	case ';', '"', '\'', '(', ')', '$', '|', '&', '>', '<', '`':
		return true
	}
	return false
}

// tokenize these inside parameter expansions
func paramOps(r rune) bool {
	switch r {
	case '}', '#', '!', ':', '-', '+', '=', '?', '%', '[', ']', '/', '^',
		',', '@':
		return true
	}
	return false
}

// these start a parameter expansion name
func paramNameOp(r rune) bool {
	switch r {
	case '}', ':', '+', '=', '%', '[', ']', '/', '^', ',', '@':
		return false
	}
	return true
}

// tokenize these inside arithmetic expansions
func arithmOps(r rune) bool {
	switch r {
	case '+', '-', '!', '*', '/', '%', '(', ')', '^', '<', '>', ':', '=',
		',', '?', '|', '&', '[', ']', '#':
		return true
	}
	return false
}

func wordBreak(r rune) bool {
	switch r {
	case ' ', '\t', '\n', ';', '&', '>', '<', '|', '(', ')', '\r':
		return true
	}
	return false
}

func (p *Parser) rune() rune {
	if p.r == '\n' {
		// p.r instead of b so that newline
		// character positions don't have col 0.
		p.npos.line++
		p.npos.col = 1
	} else {
		p.npos.col += p.w
	}
retry:
	if p.bsp < len(p.bs) {
		if b := p.bs[p.bsp]; b < utf8.RuneSelf {
			p.bsp++
			if p.litBs != nil {
				p.litBs = append(p.litBs, b)
			}
			r := rune(b)
			p.r, p.w = r, 1
			return r
		}
		if p.bsp+utf8.UTFMax >= len(p.bs) {
			// we might need up to 4 bytes to read a full
			// non-ascii rune
			p.fill()
		}
		var w int
		p.r, w = utf8.DecodeRune(p.bs[p.bsp:])
		if p.litBs != nil {
			p.litBs = append(p.litBs, p.bs[p.bsp:p.bsp+w]...)
		}
		p.bsp += w
		if p.r == utf8.RuneError && w == 1 {
			p.posErr(p.npos, "invalid UTF-8 encoding")
		}
		p.w = uint16(w)
	} else {
		if p.r == utf8.RuneSelf {
		} else if p.fill(); p.bs == nil {
			p.bsp++
			p.r = utf8.RuneSelf
		} else {
			goto retry
		}
	}
	return p.r
}

// fill reads more bytes from the input src into readBuf. Any bytes that
// had not yet been used at the end of the buffer are slid into the
// beginning of the buffer.
func (p *Parser) fill() {
	p.offs += p.bsp
	left := len(p.bs) - p.bsp
	copy(p.readBuf[:left], p.readBuf[p.bsp:])
	var n int
	var err error
	if p.readErr == nil {
		n, err = p.src.Read(p.readBuf[left:])
		p.readErr = err
	} else {
		n, err = 0, p.readErr
	}
	if n == 0 {
		// don't use p.errPass as we don't want to overwrite p.tok
		if err != nil && err != io.EOF {
			p.err = err
		}
		if left > 0 {
			p.bs = p.readBuf[:left]
		} else {
			p.bs = nil
		}
	} else {
		p.bs = p.readBuf[:left+n]
	}
	p.bsp = 0
}

func (p *Parser) nextKeepSpaces() {
	r := p.r
	p.pos = p.getPos()
	switch p.quote {
	case paramExpRepl:
		switch r {
		case '}', '/':
			p.tok = p.paramToken(r)
		case '`', '"', '$':
			p.tok = p.dqToken(r)
		default:
			p.advanceLitOther(r)
		}
	case dblQuotes:
		switch r {
		case '`', '"', '$':
			p.tok = p.dqToken(r)
		default:
			p.advanceLitDquote(r)
		}
	case hdocBody, hdocBodyTabs:
		if r == '`' || r == '$' {
			p.tok = p.dqToken(r)
		} else if p.hdocStop == nil {
			p.tok = illegalTok
		} else {
			p.advanceLitHdoc(r)
		}
	case paramExpExp:
		switch r {
		case '}':
			p.rune()
			p.tok = rightBrace
		case '`', '"', '$':
			p.tok = p.dqToken(r)
		default:
			p.advanceLitOther(r)
		}
	default: // sglQuotes
		if r == '\'' {
			p.rune()
			p.tok = sglQuote
		} else {
			p.advanceLitOther(r)
		}
	}
}

func (p *Parser) next() {
	if p.r == utf8.RuneSelf {
		p.tok = _EOF
		return
	}
	p.spaced, p.newLine = false, false
	if p.quote&allKeepSpaces != 0 {
		p.nextKeepSpaces()
		return
	}
	r := p.r
skipSpace:
	for {
		switch r {
		case utf8.RuneSelf:
			p.tok = _EOF
			return
		case ' ', '\t', '\r':
			p.spaced = true
			r = p.rune()
		case '\n':
			if p.quote == arithmExprLet || p.quote == hdocWord {
				p.tok = illegalTok
				return
			}
			p.spaced, p.newLine = true, true
			r = p.rune()
			if len(p.heredocs) > p.buriedHdocs {
				if p.doHeredocs(); p.tok == _EOF {
					return
				}
				r = p.r
			}
		case '\\':
			if !p.peekByte('\n') {
				break skipSpace
			}
			p.rune()
			r = p.rune()
		default:
			break skipSpace
		}
	}
	p.pos = p.getPos()
	switch {
	case p.quote&allRegTokens != 0:
		switch r {
		case ';', '"', '\'', '(', ')', '$', '|', '&', '>', '<', '`':
			p.tok = p.regToken(r)
		case '#':
			r = p.rune()
			p.newLit(r)
			for r != utf8.RuneSelf && r != '\n' {
				r = p.rune()
			}
			if p.keepComments {
				*p.curComs = append(*p.curComs, Comment{
					Hash: p.pos,
					Text: p.endLit(),
				})
			} else {
				p.litBs = nil
			}
			p.next()
		case '[':
			if p.quote == arrayElems {
				p.tok = leftBrack
				p.rune()
			} else {
				p.advanceLitNone(r)
			}
		case '?', '*', '+', '@', '!':
			if p.peekByte('(') {
				switch r {
				case '?':
					p.tok = globQuest
				case '*':
					p.tok = globStar
				case '+':
					p.tok = globPlus
				case '@':
					p.tok = globAt
				default: // '!'
					p.tok = globExcl
				}
				p.rune()
				p.rune()
			} else {
				p.advanceLitNone(r)
			}
		default:
			p.advanceLitNone(r)
		}
	case p.quote&allArithmExpr != 0 && arithmOps(r):
		p.tok = p.arithmToken(r)
	case p.quote&allParamExp != 0 && paramOps(r):
		p.tok = p.paramToken(r)
	case p.quote == testRegexp:
		if regOps(r) && r != '(' {
			p.tok = p.regToken(r)
		} else {
			p.advanceLitRe(r)
		}
	case regOps(r):
		p.tok = p.regToken(r)
	default:
		p.advanceLitOther(r)
	}
	if p.err != nil && p.tok != _EOF {
		p.tok = _EOF
	}
}

func (p *Parser) peekByte(b byte) bool {
	if p.bsp == len(p.bs) && p.readErr == nil {
		p.fill()
	}
	return p.bsp < len(p.bs) && p.bs[p.bsp] == b
}

func (p *Parser) regToken(r rune) token {
	switch r {
	case '\'':
		p.rune()
		return sglQuote
	case '"':
		p.rune()
		return dblQuote
	case '`':
		p.rune()
		return bckQuote
	case '&':
		switch p.rune() {
		case '&':
			p.rune()
			return andAnd
		case '>':
			if p.lang == LangPOSIX {
				break
			}
			if p.rune() == '>' {
				p.rune()
				return appAll
			}
			return rdrAll
		}
		return and
	case '|':
		switch p.rune() {
		case '|':
			p.rune()
			return orOr
		case '&':
			if p.lang == LangPOSIX {
				break
			}
			p.rune()
			return orAnd
		}
		return or
	case '$':
		switch p.rune() {
		case '\'':
			if p.lang == LangPOSIX {
				break
			}
			p.rune()
			return dollSglQuote
		case '"':
			if p.lang == LangPOSIX {
				break
			}
			p.rune()
			return dollDblQuote
		case '{':
			p.rune()
			return dollBrace
		case '[':
			if p.lang != LangBash {
				break
			}
			p.rune()
			return dollBrack
		case '(':
			if p.rune() == '(' {
				p.rune()
				return dollDblParen
			}
			return dollParen
		}
		return dollar
	case '(':
		if p.rune() == '(' && p.lang != LangPOSIX {
			p.rune()
			return dblLeftParen
		}
		return leftParen
	case ')':
		p.rune()
		return rightParen
	case ';':
		switch p.rune() {
		case ';':
			if p.rune() == '&' && p.lang == LangBash {
				p.rune()
				return dblSemiAnd
			}
			return dblSemicolon
		case '&':
			if p.lang == LangPOSIX {
				break
			}
			p.rune()
			return semiAnd
		case '|':
			if p.lang != LangMirBSDKorn {
				break
			}
			p.rune()
			return semiOr
		}
		return semicolon
	case '<':
		switch p.rune() {
		case '<':
			if r = p.rune(); r == '-' {
				p.rune()
				return dashHdoc
			} else if r == '<' && p.lang != LangPOSIX {
				p.rune()
				return wordHdoc
			}
			return hdoc
		case '>':
			p.rune()
			return rdrInOut
		case '&':
			p.rune()
			return dplIn
		case '(':
			if p.lang != LangBash {
				break
			}
			p.rune()
			return cmdIn
		}
		return rdrIn
	default: // '>'
		switch p.rune() {
		case '>':
			p.rune()
			return appOut
		case '&':
			p.rune()
			return dplOut
		case '|':
			p.rune()
			return clbOut
		case '(':
			if p.lang != LangBash {
				break
			}
			p.rune()
			return cmdOut
		}
		return rdrOut
	}
}

func (p *Parser) dqToken(r rune) token {
	switch r {
	case '"':
		p.rune()
		return dblQuote
	case '`':
		p.rune()
		return bckQuote
	default: // '$'
		switch p.rune() {
		case '{':
			p.rune()
			return dollBrace
		case '[':
			if p.lang != LangBash {
				break
			}
			p.rune()
			return dollBrack
		case '(':
			if p.rune() == '(' {
				p.rune()
				return dollDblParen
			}
			return dollParen
		}
		return dollar
	}
}

func (p *Parser) paramToken(r rune) token {
	switch r {
	case '}':
		p.rune()
		return rightBrace
	case ':':
		switch p.rune() {
		case '+':
			p.rune()
			return colPlus
		case '-':
			p.rune()
			return colMinus
		case '?':
			p.rune()
			return colQuest
		case '=':
			p.rune()
			return colAssgn
		}
		return colon
	case '+':
		p.rune()
		return plus
	case '-':
		p.rune()
		return minus
	case '?':
		p.rune()
		return quest
	case '=':
		p.rune()
		return assgn
	case '%':
		if p.rune() == '%' {
			p.rune()
			return dblPerc
		}
		return perc
	case '#':
		if p.rune() == '#' {
			p.rune()
			return dblHash
		}
		return hash
	case '!':
		p.rune()
		return exclMark
	case '[':
		p.rune()
		return leftBrack
	case '/':
		if p.rune() == '/' && p.quote != paramExpRepl {
			p.rune()
			return dblSlash
		}
		return slash
	case '^':
		if p.rune() == '^' {
			p.rune()
			return dblCaret
		}
		return caret
	case ',':
		if p.rune() == ',' {
			p.rune()
			return dblComma
		}
		return comma
	default: // '@'
		p.rune()
		return at
	}
}

func (p *Parser) arithmToken(r rune) token {
	switch r {
	case '!':
		if p.rune() == '=' {
			p.rune()
			return nequal
		}
		return exclMark
	case '=':
		if p.rune() == '=' {
			p.rune()
			return equal
		}
		return assgn
	case '(':
		p.rune()
		return leftParen
	case ')':
		p.rune()
		return rightParen
	case '&':
		switch p.rune() {
		case '&':
			p.rune()
			return andAnd
		case '=':
			p.rune()
			return andAssgn
		}
		return and
	case '|':
		switch p.rune() {
		case '|':
			p.rune()
			return orOr
		case '=':
			p.rune()
			return orAssgn
		}
		return or
	case '<':
		switch p.rune() {
		case '<':
			if p.rune() == '=' {
				p.rune()
				return shlAssgn
			}
			return hdoc
		case '=':
			p.rune()
			return lequal
		}
		return rdrIn
	case '>':
		switch p.rune() {
		case '>':
			if p.rune() == '=' {
				p.rune()
				return shrAssgn
			}
			return appOut
		case '=':
			p.rune()
			return gequal
		}
		return rdrOut
	case '+':
		switch p.rune() {
		case '+':
			p.rune()
			return addAdd
		case '=':
			p.rune()
			return addAssgn
		}
		return plus
	case '-':
		switch p.rune() {
		case '-':
			p.rune()
			return subSub
		case '=':
			p.rune()
			return subAssgn
		}
		return minus
	case '%':
		if p.rune() == '=' {
			p.rune()
			return remAssgn
		}
		return perc
	case '*':
		switch p.rune() {
		case '*':
			p.rune()
			return power
		case '=':
			p.rune()
			return mulAssgn
		}
		return star
	case '/':
		if p.rune() == '=' {
			p.rune()
			return quoAssgn
		}
		return slash
	case '^':
		if p.rune() == '=' {
			p.rune()
			return xorAssgn
		}
		return caret
	case '[':
		p.rune()
		return leftBrack
	case ']':
		p.rune()
		return rightBrack
	case ',':
		p.rune()
		return comma
	case '?':
		p.rune()
		return quest
	case ':':
		p.rune()
		return colon
	default: // '#'
		p.rune()
		return hash
	}
}

func (p *Parser) newLit(r rune) {
	// don't let r == utf8.RuneSelf go to the second case as RuneLen
	// would return -1
	if r <= utf8.RuneSelf {
		p.litBs = p.litBuf[:1]
		p.litBs[0] = byte(r)
	} else if p.bsp <= len(p.bs) {
		w := utf8.RuneLen(r)
		p.litBs = append(p.litBuf[:0], p.bs[p.bsp-w:p.bsp]...)
	}
}

func (p *Parser) discardLit(n int) { p.litBs = p.litBs[:len(p.litBs)-n] }

func (p *Parser) endLit() (s string) {
	if p.r == utf8.RuneSelf {
		s = string(p.litBs)
	} else if len(p.litBs) > 0 {
		s = string(p.litBs[:len(p.litBs)-1])
	}
	p.litBs = nil
	return
}

func (p *Parser) advanceLitOther(r rune) {
	tok := _LitWord
loop:
	for p.newLit(r); r != utf8.RuneSelf; r = p.rune() {
		switch r {
		case '\\': // escaped byte follows
			if r = p.rune(); r == '\n' {
				p.discardLit(2)
			}
		case '\n':
			switch p.quote {
			case sglQuotes, paramExpRepl, paramExpExp:
			default:
				break loop
			}
		case '\'':
			switch p.quote {
			case paramExpExp, paramExpRepl:
			default:
				break loop
			}
		case '"', '`', '$':
			if p.quote != sglQuotes {
				tok = _Lit
				break loop
			}
		case '}':
			if p.quote&allParamExp != 0 {
				break loop
			}
		case '/':
			if p.quote&allParamExp != 0 && p.quote != paramExpExp {
				break loop
			}
		case ']':
			if p.quote&allRbrack != 0 {
				break loop
			}
		case '!', '*':
			if p.quote&allArithmExpr != 0 {
				break loop
			}
			if p.quote == paramName && p.peekByte('(') {
				tok = _Lit
				break loop
			}
		case '?':
			if p.quote == paramName && p.peekByte('(') {
				tok = _Lit
				break loop
			}
			fallthrough
		case ':', '=', '%', '^', ',':
			if p.quote&allArithmExpr != 0 || p.quote&allParamReg != 0 {
				break loop
			}
		case '@':
			if p.quote == paramName && p.peekByte('(') {
				tok = _Lit
				break loop
			}
			fallthrough
		case '#', '[':
			if p.quote&allParamReg != 0 {
				break loop
			}
			if r == '[' && p.lang != LangPOSIX && p.quote&allArithmExpr != 0 {
				break loop
			}
		case '+':
			if p.quote == paramName && p.peekByte('(') {
				tok = _Lit
				break loop
			}
			fallthrough
		case '-':
			switch p.quote {
			case paramExpExp, paramExpRepl, sglQuotes:
			default:
				break loop
			}
		case ' ', '\t', ';', '&', '>', '<', '|', '(', ')', '\r':
			switch p.quote {
			case paramExpExp, paramExpRepl, sglQuotes:
			default:
				break loop
			}
		}
	}
	p.tok, p.val = tok, p.endLit()
}

func (p *Parser) advanceLitNone(r rune) {
	p.eqlOffs = 0
	tok := _LitWord
loop:
	for p.newLit(r); r != utf8.RuneSelf; r = p.rune() {
		switch r {
		case ' ', '\t', '\n', '\r', '&', '|', ';', '(', ')':
			break loop
		case '\\': // escaped byte follows
			r = p.rune()
			switch r {
			case '\n':
				p.discardLit(2)
			case '\\':
				// TODO: also treat escaped ` and $
				// differently in backquotes
				if p.quote == subCmdBckquo {
					p.discardLit(1)
					if r = p.rune(); r == '\\' {
						p.discardLit(1)
						p.rune()
					}
				}
			}
		case '>', '<':
			if p.peekByte('(') {
				tok = _Lit
			} else {
				tok = _LitRedir
			}
			break loop
		case '`':
			if p.quote != subCmdBckquo {
				tok = _Lit
			}
			break loop
		case '"', '\'', '$':
			tok = _Lit
			break loop
		case '?', '*', '+', '@', '!':
			if p.peekByte('(') {
				tok = _Lit
				break loop
			}
		case '=':
			p.eqlOffs = len(p.litBs) - 1
		case '[':
			if p.lang != LangPOSIX && len(p.litBs) > 1 && p.litBs[0] != '[' {
				tok = _Lit
				break loop
			}
		}
	}
	p.tok, p.val = tok, p.endLit()
}

func (p *Parser) advanceLitDquote(r rune) {
	tok := _LitWord
loop:
	for p.newLit(r); r != utf8.RuneSelf; r = p.rune() {
		switch r {
		case '"':
			break loop
		case '\\': // escaped byte follows
			p.rune()
		case '`', '$':
			tok = _Lit
			break loop
		}
	}
	p.tok, p.val = tok, p.endLit()
}

func (p *Parser) advanceLitHdoc(r rune) {
	p.tok = _Lit
	p.newLit(r)
	if p.quote == hdocBodyTabs {
		for r == '\t' {
			r = p.rune()
		}
	}
	lStart := len(p.litBs) - 1
	for ; ; r = p.rune() {
		switch r {
		case '`', '$':
			p.val = p.endLit()
			return
		case '\\': // escaped byte follows
			p.rune()
		case '\n', utf8.RuneSelf:
			if bytes.HasPrefix(p.litBs[lStart:], p.hdocStop) {
				p.val = p.endLit()[:lStart]
				if p.val == "" {
					p.tok = illegalTok
				}
				p.hdocStop = nil
				return
			}
			if r == utf8.RuneSelf {
				return
			}
			if p.quote == hdocBodyTabs {
				for p.peekByte('\t') {
					p.rune()
				}
			}
			lStart = len(p.litBs)
		}
	}
}

func (p *Parser) hdocLitWord() *Word {
	r := p.r
	p.newLit(r)
	pos := p.getPos()
	for ; ; r = p.rune() {
		if r == utf8.RuneSelf {
			return nil
		}
		if p.quote == hdocBodyTabs {
			for r == '\t' {
				r = p.rune()
			}
		}
		lStart := len(p.litBs) - 1
		for r != utf8.RuneSelf && r != '\n' {
			r = p.rune()
		}
		if bytes.HasPrefix(p.litBs[lStart:], p.hdocStop) {
			p.hdocStop = nil
			val := p.endLit()[:lStart]
			if val == "" {
				return nil
			}
			return p.word(p.wps(p.lit(pos, val)))
		}
	}
}

func (p *Parser) advanceLitRe(r rune) {
	lparens := 0
loop:
	for p.newLit(r); r != utf8.RuneSelf; r = p.rune() {
		switch r {
		case '\\':
			p.rune()
		case '(':
			lparens++
		case ')':
			lparens--
		case ' ', '\t', '\r', '\n', ';':
			if lparens == 0 {
				break loop
			}
		}
	}
	p.tok, p.val = _LitWord, p.endLit()
}

func testUnaryOp(val string) UnTestOperator {
	switch val {
	case "!":
		return TsNot
	case "-e", "-a":
		return TsExists
	case "-f":
		return TsRegFile
	case "-d":
		return TsDirect
	case "-c":
		return TsCharSp
	case "-b":
		return TsBlckSp
	case "-p":
		return TsNmPipe
	case "-S":
		return TsSocket
	case "-L", "-h":
		return TsSmbLink
	case "-k":
		return TsSticky
	case "-g":
		return TsGIDSet
	case "-u":
		return TsUIDSet
	case "-G":
		return TsGrpOwn
	case "-O":
		return TsUsrOwn
	case "-N":
		return TsModif
	case "-r":
		return TsRead
	case "-w":
		return TsWrite
	case "-x":
		return TsExec
	case "-s":
		return TsNoEmpty
	case "-t":
		return TsFdTerm
	case "-z":
		return TsEmpStr
	case "-n":
		return TsNempStr
	case "-o":
		return TsOptSet
	case "-v":
		return TsVarSet
	case "-R":
		return TsRefVar
	default:
		return 0
	}
}

func testBinaryOp(val string) BinTestOperator {
	switch val {
	case "==", "=":
		return TsMatch
	case "!=":
		return TsNoMatch
	case "=~":
		return TsReMatch
	case "-nt":
		return TsNewer
	case "-ot":
		return TsOlder
	case "-ef":
		return TsDevIno
	case "-eq":
		return TsEql
	case "-ne":
		return TsNeq
	case "-le":
		return TsLeq
	case "-ge":
		return TsGeq
	case "-lt":
		return TsLss
	case "-gt":
		return TsGtr
	default:
		return 0
	}
}
