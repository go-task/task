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
		',', '@', '*':
		return true
	}
	return false
}

// these start a parameter expansion name
func paramNameOp(r rune) bool {
	switch r {
	case '}', ':', '+', '=', '%', '[', ']', '/', '^', ',':
		return false
	}
	return true
}

// tokenize these inside arithmetic expansions
func arithmOps(r rune) bool {
	switch r {
	case '+', '-', '!', '~', '*', '/', '%', '(', ')', '^', '<', '>', ':', '=',
		',', '?', '|', '&', '[', ']', '#':
		return true
	}
	return false
}

func bquoteEscaped(b byte) bool {
	switch b {
	case '$', '`', '\\':
		return true
	}
	return false
}

const escNewl rune = utf8.RuneSelf + 1

func (p *Parser) rune() rune {
	if p.r == '\n' || p.r == escNewl {
		// p.r instead of b so that newline
		// character positions don't have col 0.
		p.npos.line++
		p.npos.col = 0
	}
	p.npos.col += p.w
	bquotes := 0
retry:
	if p.bsp < len(p.bs) {
		if b := p.bs[p.bsp]; b < utf8.RuneSelf {
			p.bsp++
			if b == '\\' {
				if p.r != '\\' && p.peekByte('\n') {
					p.bsp++
					p.w, p.r = 1, escNewl
					return escNewl
				}
				if p.openBquotes > 0 && bquotes < p.openBquotes &&
					p.bsp < len(p.bs) && bquoteEscaped(p.bs[p.bsp]) {
					bquotes++
					goto retry
				}
			}
			if b == '`' {
				p.lastBquoteEsc = bquotes
			}
			if p.litBs != nil {
				p.litBs = append(p.litBs, b)
			}
			p.w, p.r = 1, rune(b)
			return p.r
		}
		if !utf8.FullRune(p.bs[p.bsp:]) {
			// we need more bytes to read a full non-ascii rune
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
			p.w = 1
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
readAgain:
	n, err := 0, p.readErr
	if err == nil {
		n, err = p.src.Read(p.readBuf[left:])
		p.readErr = err
	}
	if n == 0 {
		if err == nil {
			goto readAgain
		}
		// don't use p.errPass as we don't want to overwrite p.tok
		if err != io.EOF {
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
			p.tok = p.regToken(r)
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
		switch {
		case r == '`' || r == '$':
			p.tok = p.dqToken(r)
		case p.hdocStop == nil:
			p.tok = _Newl
		default:
			p.advanceLitHdoc(r)
		}
	default: // paramExpExp:
		switch r {
		case '}':
			p.tok = p.paramToken(r)
		case '`', '"', '$', '\'':
			p.tok = p.regToken(r)
		default:
			p.advanceLitOther(r)
		}
	}
	if p.err != nil && p.tok != _EOF {
		p.tok = _EOF
	}
}

func (p *Parser) next() {
	if p.r == utf8.RuneSelf {
		p.tok = _EOF
		return
	}
	for p.r == escNewl {
		p.rune()
	}
	p.spaced = false
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
		case escNewl:
			r = p.rune()
		case ' ', '\t', '\r':
			p.spaced = true
			r = p.rune()
		case '\n':
			if p.tok == _Newl {
				// merge consecutive newline tokens
				r = p.rune()
				continue
			}
			p.spaced = true
			p.tok = _Newl
			if p.quote != hdocWord && len(p.heredocs) > p.buriedHdocs {
				p.doHeredocs()
			}
			return
		default:
			break skipSpace
		}
	}
	if p.stopAt != nil && (p.spaced || p.tok == illegalTok || stopToken(p.tok)) {
		w := utf8.RuneLen(r)
		if bytes.HasPrefix(p.bs[p.bsp-w:], p.stopAt) {
			p.r = utf8.RuneSelf
			p.w = 1
			p.tok = _EOF
			return
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
			for r != '\n' && r != utf8.RuneSelf {
				if r == escNewl {
					p.litBs = append(p.litBs, '\\', '\n')
					break
				}
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
		case '[', '=':
			if p.quote == arrayElems {
				p.tok = p.paramToken(r)
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
		if !p.rxFirstPart && p.spaced {
			p.quote = noState
			goto skipSpace
		}
		p.rxFirstPart = false
		switch r {
		case ';', '"', '\'', '$', '&', '>', '<', '`':
			p.tok = p.regToken(r)
		case ')':
			if p.rxOpenParens > 0 {
				// continuation of open paren
				p.advanceLitRe(r)
			} else {
				p.tok = rightParen
				p.quote = noState
				p.rune() // we are tokenizing manually
			}
		default: // including '(', '|'
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
	if p.bsp == len(p.bs) {
		p.fill()
	}
	return p.bsp < len(p.bs) && p.bs[p.bsp] == b
}

func (p *Parser) regToken(r rune) token {
	switch r {
	case '\'':
		if p.openBquotes > 0 {
			// bury openBquotes
			p.buriedBquotes = p.openBquotes
			p.openBquotes = 0
		}
		p.rune()
		return sglQuote
	case '"':
		p.rune()
		return dblQuote
	case '`':
		// Don't call p.rune, as we need to work out p.openBquotes to
		// properly handle backslashes in the lexer.
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
			if p.lang != LangBash || p.quote == paramExpName {
				// latter to not tokenise ${$[@]} as $[
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
		// Don't call p.rune, as we need to work out p.openBquotes to
		// properly handle backslashes in the lexer.
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
	case ']':
		p.rune()
		return rightBrack
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
	case '@':
		p.rune()
		return at
	default: // '*'
		p.rune()
		return star
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
	case '~':
		p.rune()
		return tilde
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
	switch {
	case r < utf8.RuneSelf:
		p.litBs = p.litBuf[:1]
		p.litBs[0] = byte(r)
	case r > escNewl:
		w := utf8.RuneLen(r)
		p.litBs = append(p.litBuf[:0], p.bs[p.bsp-w:p.bsp]...)
	default:
		// don't let r == utf8.RuneSelf go to the second case as RuneLen
		// would return -1
		p.litBs = p.litBuf[:0]
	}
}

func (p *Parser) endLit() (s string) {
	if p.r == utf8.RuneSelf || p.r == escNewl {
		s = string(p.litBs)
	} else {
		s = string(p.litBs[:len(p.litBs)-int(p.w)])
	}
	p.litBs = nil
	return
}

func (p *Parser) isLitRedir() bool {
	lit := p.litBs[:len(p.litBs)-1]
	if lit[0] == '{' && lit[len(lit)-1] == '}' {
		return ValidName(string(lit[1 : len(lit)-1]))
	}
	for _, b := range lit {
		switch b {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		default:
			return false
		}
	}
	return true
}

func (p *Parser) advanceNameCont(r rune) {
	// we know that r is a letter or underscore
loop:
	for p.newLit(r); r != utf8.RuneSelf; r = p.rune() {
		switch {
		case 'a' <= r && r <= 'z':
		case 'A' <= r && r <= 'Z':
		case r == '_':
		case '0' <= r && r <= '9':
		case r == escNewl:
		default:
			break loop
		}
	}
	p.tok, p.val = _LitWord, p.endLit()
}

func (p *Parser) advanceLitOther(r rune) {
	tok := _LitWord
loop:
	for p.newLit(r); r != utf8.RuneSelf; r = p.rune() {
		switch r {
		case '\\': // escaped byte follows
			p.rune()
		case '"', '`', '$':
			tok = _Lit
			break loop
		case '}':
			if p.quote&allParamExp != 0 {
				break loop
			}
		case '/':
			if p.quote != paramExpExp {
				break loop
			}
		case ':', '=', '%', '^', ',', '?', '!', '~', '*':
			if p.quote&allArithmExpr != 0 || p.quote == paramExpName {
				break loop
			}
		case '[', ']':
			if p.lang != LangPOSIX && p.quote&allArithmExpr != 0 {
				break loop
			}
			fallthrough
		case '#', '@':
			if p.quote&allParamReg != 0 {
				break loop
			}
		case '\'', '+', '-', ' ', '\t', ';', '&', '>', '<', '|', '(', ')', '\n', '\r':
			if p.quote&allKeepSpaces == 0 {
				break loop
			}
		}
	}
	p.tok, p.val = tok, p.endLit()
}

func (p *Parser) advanceLitNone(r rune) {
	p.eqlOffs = -1
	tok := _LitWord
loop:
	for p.newLit(r); r != utf8.RuneSelf; r = p.rune() {
		switch r {
		case ' ', '\t', '\n', '\r', '&', '|', ';', '(', ')':
			break loop
		case '\\': // escaped byte follows
			p.rune()
		case '>', '<':
			if p.peekByte('(') {
				tok = _Lit
			} else if p.isLitRedir() {
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
			if p.eqlOffs < 0 {
				p.eqlOffs = len(p.litBs) - 1
			}
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
		case escNewl, '`', '$':
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
		case escNewl, '`', '$':
			p.val = p.endLit()
			return
		case '\\': // escaped byte follows
			p.rune()
		case '\n', utf8.RuneSelf:
			if p.parsingDoc {
				if r == utf8.RuneSelf {
					p.val = p.endLit()
					return
				}
			} else if lStart >= 0 && bytes.HasPrefix(p.litBs[lStart:], p.hdocStop) {
				p.val = p.endLit()[:lStart]
				if p.val == "" {
					p.tok = _Newl
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

func (p *Parser) quotedHdocWord() *Word {
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
			if r == escNewl {
				p.litBs = append(p.litBs, '\\', '\n')
			}
			r = p.rune()
		}
		if lStart >= 0 && bytes.HasPrefix(p.litBs[lStart:], p.hdocStop) {
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
	for p.newLit(r); ; r = p.rune() {
		switch r {
		case '\\':
			p.rune()
		case '(':
			p.rxOpenParens++
		case ')':
			if p.rxOpenParens--; p.rxOpenParens < 0 {
				p.tok, p.val = _LitWord, p.endLit()
				p.quote = noState
				return
			}
		case ' ', '\t', '\r', '\n':
			if p.rxOpenParens <= 0 {
				p.tok, p.val = _LitWord, p.endLit()
				p.quote = noState
				return
			}
		case '"', '\'', '$', '`':
			p.tok, p.val = _Lit, p.endLit()
			return
		case utf8.RuneSelf, ';', '&', '>', '<':
			p.tok, p.val = _LitWord, p.endLit()
			p.quote = noState
			return
		}
	}
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
	case "=":
		return TsMatchShort
	case "==":
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
