// Copyright (c) 2016, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package syntax

import "fmt"

// Node represents a syntax tree node.
type Node interface {
	// Pos returns the position of the first character of the node. Comments
	// are ignored, except if the node is a *File.
	Pos() Pos
	// End returns the position of the character immediately after the node.
	// If the character is a newline, the line number won't cross into the
	// next line. Comments are ignored, except if the node is a *File.
	End() Pos
}

// File represents a shell source file.
type File struct {
	Name string

	StmtList
}

// StmtList is a list of statements with any number of trailing comments. Both
// lists can be empty.
type StmtList struct {
	Stmts []*Stmt
	Last  []Comment
}

func (s StmtList) pos() Pos {
	if len(s.Stmts) > 0 {
		s := s.Stmts[0]
		sPos := s.Pos()
		if len(s.Comments) > 0 {
			if cPos := s.Comments[0].Pos(); sPos.After(cPos) {
				return cPos
			}
		}
		return sPos
	}
	if len(s.Last) > 0 {
		return s.Last[0].Pos()
	}
	return Pos{}
}

func (s StmtList) end() Pos {
	if len(s.Last) > 0 {
		return s.Last[len(s.Last)-1].End()
	}
	if len(s.Stmts) > 0 {
		s := s.Stmts[len(s.Stmts)-1]
		sEnd := s.End()
		if len(s.Comments) > 0 {
			if cEnd := s.Comments[0].End(); cEnd.After(sEnd) {
				return cEnd
			}
		}
		return sEnd
	}
	return Pos{}
}

func (s StmtList) empty() bool {
	return len(s.Stmts) == 0 && len(s.Last) == 0
}

// Pos is a position within a shell source file.
type Pos struct {
	offs      uint32
	line, col uint16
}

// Offset returns the byte offset of the position in the original source file.
// Byte offsets start at 0.
func (p Pos) Offset() uint { return uint(p.offs) }

// Line returns the line number of the position, starting at 1.
func (p Pos) Line() uint { return uint(p.line) }

// Col returns the column number of the position, starting at 1. It counts in
// bytes.
func (p Pos) Col() uint { return uint(p.col) }

func (p Pos) String() string {
	return fmt.Sprintf("%d:%d", p.Line(), p.Col())
}

// IsValid reports whether the position is valid. All positions in nodes
// returned by Parse are valid.
func (p Pos) IsValid() bool { return p.line > 0 }

// After reports whether the position p is after p2. It is a more expressive
// version of p.Offset() > p2.Offset().
func (p Pos) After(p2 Pos) bool { return p.offs > p2.offs }

func (f *File) Pos() Pos { return f.StmtList.pos() }
func (f *File) End() Pos { return f.StmtList.end() }

func posAddCol(p Pos, n int) Pos {
	p.col += uint16(n)
	p.offs += uint32(n)
	return p
}

func posMax(p1, p2 Pos) Pos {
	if p2.After(p1) {
		return p2
	}
	return p1
}

// Comment represents a single comment on a single line.
type Comment struct {
	Hash Pos
	Text string
}

func (c *Comment) Pos() Pos { return c.Hash }
func (c *Comment) End() Pos { return posAddCol(c.Hash, 1+len(c.Text)) }

// Stmt represents a statement, also known as a "complete command". It is
// compromised of a command and other components that may come before or after
// it.
type Stmt struct {
	Comments   []Comment
	Cmd        Command
	Position   Pos
	Semicolon  Pos  // position of ';', '&', or '|&', if any
	Negated    bool // ! stmt
	Background bool // stmt &
	Coprocess  bool // mksh's |&

	Redirs []*Redirect // stmt >a <b
}

func (s *Stmt) Pos() Pos { return s.Position }
func (s *Stmt) End() Pos {
	if s.Semicolon.IsValid() {
		end := posAddCol(s.Semicolon, 1) // ';' or '&'
		if s.Coprocess {
			end = posAddCol(end, 1) // '|&'
		}
		return end
	}
	end := s.Position
	if s.Negated {
		end = posAddCol(end, 1)
	}
	if s.Cmd != nil {
		end = s.Cmd.End()
	}
	if len(s.Redirs) > 0 {
		end = posMax(end, s.Redirs[len(s.Redirs)-1].End())
	}
	return end
}

// Command represents all nodes that are simple or compound commands, including
// function declarations.
//
// These are *CallExpr, *IfClause, *WhileClause, *ForClause, *CaseClause,
// *Block, *Subshell, *BinaryCmd, *FuncDecl, *ArithmCmd, *TestClause,
// *DeclClause, *LetClause, *TimeClause, and *CoprocClause.
type Command interface {
	Node
	commandNode()
}

func (*CallExpr) commandNode()     {}
func (*IfClause) commandNode()     {}
func (*WhileClause) commandNode()  {}
func (*ForClause) commandNode()    {}
func (*CaseClause) commandNode()   {}
func (*Block) commandNode()        {}
func (*Subshell) commandNode()     {}
func (*BinaryCmd) commandNode()    {}
func (*FuncDecl) commandNode()     {}
func (*ArithmCmd) commandNode()    {}
func (*TestClause) commandNode()   {}
func (*DeclClause) commandNode()   {}
func (*LetClause) commandNode()    {}
func (*TimeClause) commandNode()   {}
func (*CoprocClause) commandNode() {}

// Assign represents an assignment to a variable.
//
// Here and elsewhere, Index can mean either an index expression into an indexed
// array, or a string key into an associative array.
//
// If Index is non-nil, the value will be a word and not an array as nested
// arrays are not allowed.
//
// If Naked is true and Name is nil, the assignment is part of a DeclClause and
// the assignment expression (in the Value field) will be evaluated at run-time.
type Assign struct {
	Append bool // +=
	Naked  bool // without '='
	Name   *Lit
	Index  ArithmExpr // [i], ["k"]
	Value  *Word      // =val
	Array  *ArrayExpr // =(arr)
}

func (a *Assign) Pos() Pos {
	if a.Name == nil {
		return a.Value.Pos()
	}
	return a.Name.Pos()
}

func (a *Assign) End() Pos {
	if a.Value != nil {
		return a.Value.End()
	}
	if a.Array != nil {
		return a.Array.End()
	}
	if a.Index != nil {
		return posAddCol(a.Index.End(), 2)
	}
	if a.Naked {
		return a.Name.End()
	}
	return posAddCol(a.Name.End(), 1)
}

// Redirect represents an input/output redirection.
type Redirect struct {
	OpPos Pos
	Op    RedirOperator
	N     *Lit  // fd>, or {varname}> in Bash
	Word  *Word // >word
	Hdoc  *Word // here-document body
}

func (r *Redirect) Pos() Pos {
	if r.N != nil {
		return r.N.Pos()
	}
	return r.OpPos
}
func (r *Redirect) End() Pos { return r.Word.End() }

// CallExpr represents a command execution or function call, otherwise known as
// a "simple command".
//
// If Args is empty, Assigns apply to the shell environment. Otherwise, they are
// variables that cannot be arrays and which only apply to the call.
type CallExpr struct {
	Assigns []*Assign // a=x b=y args
	Args    []*Word
}

func (c *CallExpr) Pos() Pos {
	if len(c.Assigns) > 0 {
		return c.Assigns[0].Pos()
	}
	return c.Args[0].Pos()
}

func (c *CallExpr) End() Pos {
	if len(c.Args) == 0 {
		return c.Assigns[len(c.Assigns)-1].End()
	}
	return c.Args[len(c.Args)-1].End()
}

// Subshell represents a series of commands that should be executed in a nested
// shell environment.
type Subshell struct {
	Lparen, Rparen Pos
	StmtList
}

func (s *Subshell) Pos() Pos { return s.Lparen }
func (s *Subshell) End() Pos { return posAddCol(s.Rparen, 1) }

// Block represents a series of commands that should be executed in a nested
// scope.
type Block struct {
	Lbrace, Rbrace Pos
	StmtList
}

func (b *Block) Pos() Pos { return b.Lbrace }
func (b *Block) End() Pos { return posAddCol(b.Rbrace, 1) }

// IfClause represents an if statement.
type IfClause struct {
	Elif    bool // whether this IfClause begins with "elif"
	IfPos   Pos  // position of the starting "if" or "elif" token
	ThenPos Pos
	ElsePos Pos // position of a following "else" or "elif", if any
	FiPos   Pos // position of "fi", empty if Elif == true

	Cond StmtList
	Then StmtList
	Else StmtList

	ElseComments []Comment // comments on the "else"
}

func (c *IfClause) Pos() Pos { return c.IfPos }
func (c *IfClause) End() Pos {
	if !c.FiPos.IsValid() {
		return posAddCol(c.ElsePos, 4)
	}
	return posAddCol(c.FiPos, 2)
}

// FollowedByElif reports whether this IfClause is followed by an "elif"
// IfClause in its Else branch. This is true if Else.Stmts has exactly one
// statement with an IfClause whose Elif field is true.
func (c *IfClause) FollowedByElif() bool {
	if len(c.Else.Stmts) != 1 {
		return false
	}
	ic, _ := c.Else.Stmts[0].Cmd.(*IfClause)
	return ic != nil && ic.Elif
}

func (c *IfClause) bodyEndPos() Pos {
	if c.ElsePos.IsValid() {
		return c.ElsePos
	}
	return c.FiPos
}

// WhileClause represents a while or an until clause.
type WhileClause struct {
	WhilePos, DoPos, DonePos Pos
	Until                    bool
	Cond                     StmtList
	Do                       StmtList
}

func (w *WhileClause) Pos() Pos { return w.WhilePos }
func (w *WhileClause) End() Pos { return posAddCol(w.DonePos, 4) }

// ForClause represents a for or a select clause. The latter is only present in
// Bash.
type ForClause struct {
	ForPos, DoPos, DonePos Pos
	Select                 bool
	Loop                   Loop
	Do                     StmtList
}

func (f *ForClause) Pos() Pos { return f.ForPos }
func (f *ForClause) End() Pos { return posAddCol(f.DonePos, 4) }

// Loop holds either *WordIter or *CStyleLoop.
type Loop interface {
	Node
	loopNode()
}

func (*WordIter) loopNode()   {}
func (*CStyleLoop) loopNode() {}

// WordIter represents the iteration of a variable over a series of words in a
// for clause.
type WordIter struct {
	Name  *Lit
	Items []*Word
}

func (w *WordIter) Pos() Pos { return w.Name.Pos() }
func (w *WordIter) End() Pos { return posMax(w.Name.End(), wordLastEnd(w.Items)) }

// CStyleLoop represents the behaviour of a for clause similar to the C
// language.
//
// This node will only appear with LangBash.
type CStyleLoop struct {
	Lparen, Rparen   Pos
	Init, Cond, Post ArithmExpr
}

func (c *CStyleLoop) Pos() Pos { return c.Lparen }
func (c *CStyleLoop) End() Pos { return posAddCol(c.Rparen, 2) }

// BinaryCmd represents a binary expression between two statements.
type BinaryCmd struct {
	OpPos Pos
	Op    BinCmdOperator
	X, Y  *Stmt
}

func (b *BinaryCmd) Pos() Pos { return b.X.Pos() }
func (b *BinaryCmd) End() Pos { return b.Y.End() }

// FuncDecl represents the declaration of a function.
type FuncDecl struct {
	Position Pos
	RsrvWord bool // non-posix "function f()" style
	Name     *Lit
	Body     *Stmt
}

func (f *FuncDecl) Pos() Pos { return f.Position }
func (f *FuncDecl) End() Pos { return f.Body.End() }

// Word represents a shell word, containing one or more word parts contiguous to
// each other. The word is delimeted by word boundaries, such as spaces,
// newlines, semicolons, or parentheses.
type Word struct {
	Parts []WordPart
}

func (w *Word) Pos() Pos { return w.Parts[0].Pos() }
func (w *Word) End() Pos { return w.Parts[len(w.Parts)-1].End() }

// WordPart represents all nodes that can form part of a word.
//
// These are *Lit, *SglQuoted, *DblQuoted, *ParamExp, *CmdSubst, *ArithmExp,
// *ProcSubst, and *ExtGlob.
type WordPart interface {
	Node
	wordPartNode()
}

func (*Lit) wordPartNode()       {}
func (*SglQuoted) wordPartNode() {}
func (*DblQuoted) wordPartNode() {}
func (*ParamExp) wordPartNode()  {}
func (*CmdSubst) wordPartNode()  {}
func (*ArithmExp) wordPartNode() {}
func (*ProcSubst) wordPartNode() {}
func (*ExtGlob) wordPartNode()   {}

// Lit represents a string literal.
//
// Note that a parsed string literal may not appear as-is in the original source
// code, as it is possible to split literals by escaping newlines. The splitting
// is lost, but the end position is not.
type Lit struct {
	ValuePos, ValueEnd Pos
	Value              string
}

func (l *Lit) Pos() Pos { return l.ValuePos }
func (l *Lit) End() Pos { return l.ValueEnd }

// SglQuoted represents a string within single quotes.
type SglQuoted struct {
	Left, Right Pos
	Dollar      bool // $''
	Value       string
}

func (q *SglQuoted) Pos() Pos { return q.Left }
func (q *SglQuoted) End() Pos { return posAddCol(q.Right, 1) }

// DblQuoted represents a list of nodes within double quotes.
type DblQuoted struct {
	Position Pos
	Dollar   bool // $""
	Parts    []WordPart
}

func (q *DblQuoted) Pos() Pos { return q.Position }
func (q *DblQuoted) End() Pos {
	if len(q.Parts) == 0 {
		if q.Dollar {
			return posAddCol(q.Position, 3)
		}
		return posAddCol(q.Position, 2)
	}
	return posAddCol(q.Parts[len(q.Parts)-1].End(), 1)
}

// CmdSubst represents a command substitution.
type CmdSubst struct {
	Left, Right Pos
	StmtList

	TempFile bool // mksh's ${ foo;}
	ReplyVar bool // mksh's ${|foo;}
}

func (c *CmdSubst) Pos() Pos { return c.Left }
func (c *CmdSubst) End() Pos { return posAddCol(c.Right, 1) }

// ParamExp represents a parameter expansion.
type ParamExp struct {
	Dollar, Rbrace Pos
	Short          bool // $a instead of ${a}
	Excl           bool // ${!a}
	Length         bool // ${#a}
	Width          bool // ${%a}
	Param          *Lit
	Index          ArithmExpr       // ${a[i]}, ${a["k"]}
	Slice          *Slice           // ${a:x:y}
	Repl           *Replace         // ${a/x/y}
	Names          ParNamesOperator // ${!prefix*} or ${!prefix@}
	Exp            *Expansion       // ${a:-b}, ${a#b}, etc
}

func (p *ParamExp) Pos() Pos { return p.Dollar }
func (p *ParamExp) End() Pos {
	if !p.Short {
		return posAddCol(p.Rbrace, 1)
	}
	if p.Index != nil {
		return posAddCol(p.Index.End(), 1)
	}
	return p.Param.End()
}

func (p *ParamExp) nakedIndex() bool {
	return p.Short && p.Index != nil
}

// Slice represents a character slicing expression inside a ParamExp.
//
// This node will only appear in LangBash and LangMirBSDKorn.
type Slice struct {
	Offset, Length ArithmExpr
}

// Replace represents a search and replace expression inside a ParamExp.
type Replace struct {
	All        bool
	Orig, With *Word
}

// Expansion represents string manipulation in a ParamExp other than those
// covered by Replace.
type Expansion struct {
	Op   ParExpOperator
	Word *Word
}

// ArithmExp represents an arithmetic expansion.
type ArithmExp struct {
	Left, Right Pos
	Bracket     bool // deprecated $[expr] form
	Unsigned    bool // mksh's $((# expr))
	X           ArithmExpr
}

func (a *ArithmExp) Pos() Pos { return a.Left }
func (a *ArithmExp) End() Pos {
	if a.Bracket {
		return posAddCol(a.Right, 1)
	}
	return posAddCol(a.Right, 2)
}

// ArithmCmd represents an arithmetic command.
//
// This node will only appear in LangBash and LangMirBSDKorn.
type ArithmCmd struct {
	Left, Right Pos
	Unsigned    bool // mksh's ((# expr))
	X           ArithmExpr
}

func (a *ArithmCmd) Pos() Pos { return a.Left }
func (a *ArithmCmd) End() Pos { return posAddCol(a.Right, 2) }

// ArithmExpr represents all nodes that form arithmetic expressions.
//
// These are *BinaryArithm, *UnaryArithm, *ParenArithm, and *Word.
type ArithmExpr interface {
	Node
	arithmExprNode()
}

func (*BinaryArithm) arithmExprNode() {}
func (*UnaryArithm) arithmExprNode()  {}
func (*ParenArithm) arithmExprNode()  {}
func (*Word) arithmExprNode()         {}

// BinaryArithm represents a binary arithmetic expression.
//
// If Op is any assign operator, X will be a word with a single *Lit whose value
// is a valid name.
//
// Ternary operators like "a ? b : c" are fit into this structure. Thus, if
// Op==Quest, Y will be a *BinaryArithm with Op==Colon. Op can only be Colon in
// that scenario.
type BinaryArithm struct {
	OpPos Pos
	Op    BinAritOperator
	X, Y  ArithmExpr
}

func (b *BinaryArithm) Pos() Pos { return b.X.Pos() }
func (b *BinaryArithm) End() Pos { return b.Y.End() }

// UnaryArithm represents an unary arithmetic expression. The unary opearator
// may come before or after the sub-expression.
//
// If Op is Inc or Dec, X will be a word with a single *Lit whose value is a
// valid name.
type UnaryArithm struct {
	OpPos Pos
	Op    UnAritOperator
	Post  bool
	X     ArithmExpr
}

func (u *UnaryArithm) Pos() Pos {
	if u.Post {
		return u.X.Pos()
	}
	return u.OpPos
}

func (u *UnaryArithm) End() Pos {
	if u.Post {
		return posAddCol(u.OpPos, 2)
	}
	return u.X.End()
}

// ParenArithm represents an arithmetic expression within parentheses.
type ParenArithm struct {
	Lparen, Rparen Pos
	X              ArithmExpr
}

func (p *ParenArithm) Pos() Pos { return p.Lparen }
func (p *ParenArithm) End() Pos { return posAddCol(p.Rparen, 1) }

// CaseClause represents a case (switch) clause.
type CaseClause struct {
	Case, Esac Pos
	Word       *Word
	Items      []*CaseItem
	Last       []Comment
}

func (c *CaseClause) Pos() Pos { return c.Case }
func (c *CaseClause) End() Pos { return posAddCol(c.Esac, 4) }

// CaseItem represents a pattern list (case) within a CaseClause.
type CaseItem struct {
	Op       CaseOperator
	OpPos    Pos // unset if it was finished by "esac"
	Comments []Comment
	Patterns []*Word
	StmtList
}

func (c *CaseItem) Pos() Pos { return c.Patterns[0].Pos() }
func (c *CaseItem) End() Pos {
	if c.OpPos.IsValid() {
		return posAddCol(c.OpPos, len(c.Op.String()))
	}
	return c.StmtList.end()
}

// TestClause represents a Bash extended test clause.
//
// This node will only appear in LangBash and LangMirBSDKorn.
type TestClause struct {
	Left, Right Pos
	X           TestExpr
}

func (t *TestClause) Pos() Pos { return t.Left }
func (t *TestClause) End() Pos { return posAddCol(t.Right, 2) }

// TestExpr represents all nodes that form test expressions.
//
// These are *BinaryTest, *UnaryTest, *ParenTest, and *Word.
type TestExpr interface {
	Node
	testExprNode()
}

func (*BinaryTest) testExprNode() {}
func (*UnaryTest) testExprNode()  {}
func (*ParenTest) testExprNode()  {}
func (*Word) testExprNode()       {}

// BinaryTest represents a binary test expression.
type BinaryTest struct {
	OpPos Pos
	Op    BinTestOperator
	X, Y  TestExpr
}

func (b *BinaryTest) Pos() Pos { return b.X.Pos() }
func (b *BinaryTest) End() Pos { return b.Y.End() }

// UnaryTest represents a unary test expression. The unary opearator may come
// before or after the sub-expression.
type UnaryTest struct {
	OpPos Pos
	Op    UnTestOperator
	X     TestExpr
}

func (u *UnaryTest) Pos() Pos { return u.OpPos }
func (u *UnaryTest) End() Pos { return u.X.End() }

// ParenTest represents a test expression within parentheses.
type ParenTest struct {
	Lparen, Rparen Pos
	X              TestExpr
}

func (p *ParenTest) Pos() Pos { return p.Lparen }
func (p *ParenTest) End() Pos { return posAddCol(p.Rparen, 1) }

// DeclClause represents a Bash declare clause.
//
// This node will only appear with LangBash.
type DeclClause struct {
	// Variant is one of "declare", "local", "export", "readonly",
	// "typeset", or "nameref".
	Variant *Lit
	Opts    []*Word
	Assigns []*Assign
}

func (d *DeclClause) Pos() Pos { return d.Variant.Pos() }
func (d *DeclClause) End() Pos {
	if len(d.Assigns) > 0 {
		return d.Assigns[len(d.Assigns)-1].End()
	}
	if len(d.Opts) > 0 {
		return wordLastEnd(d.Opts)
	}
	return d.Variant.End()
}

// ArrayExpr represents a Bash array expression.
//
// This node will only appear with LangBash.
type ArrayExpr struct {
	Lparen, Rparen Pos
	Elems          []*ArrayElem
	Last           []Comment
}

func (a *ArrayExpr) Pos() Pos { return a.Lparen }
func (a *ArrayExpr) End() Pos { return posAddCol(a.Rparen, 1) }

// ArrayElem represents a Bash array element.
type ArrayElem struct {
	Index    ArithmExpr // [i]=, ["k"]=
	Value    *Word
	Comments []Comment
}

func (a *ArrayElem) Pos() Pos {
	if a.Index != nil {
		return a.Index.Pos()
	}
	return a.Value.Pos()
}
func (a *ArrayElem) End() Pos { return a.Value.End() }

// ExtGlob represents a Bash extended globbing expression. Note that these are
// parsed independently of whether shopt has been called or not.
//
// This node will only appear in LangBash and LangMirBSDKorn.
type ExtGlob struct {
	OpPos   Pos
	Op      GlobOperator
	Pattern *Lit
}

func (e *ExtGlob) Pos() Pos { return e.OpPos }
func (e *ExtGlob) End() Pos { return posAddCol(e.Pattern.End(), 1) }

// ProcSubst represents a Bash process substitution.
//
// This node will only appear with LangBash.
type ProcSubst struct {
	OpPos, Rparen Pos
	Op            ProcOperator
	StmtList
}

func (s *ProcSubst) Pos() Pos { return s.OpPos }
func (s *ProcSubst) End() Pos { return posAddCol(s.Rparen, 1) }

// TimeClause represents a Bash time clause. PosixFormat corresponds to the -p
// flag.
//
// This node will only appear in LangBash and LangMirBSDKorn.
type TimeClause struct {
	Time        Pos
	PosixFormat bool
	Stmt        *Stmt
}

func (c *TimeClause) Pos() Pos { return c.Time }
func (c *TimeClause) End() Pos {
	if c.Stmt == nil {
		return posAddCol(c.Time, 4)
	}
	return c.Stmt.End()
}

// CoprocClause represents a Bash coproc clause.
//
// This node will only appear with LangBash.
type CoprocClause struct {
	Coproc Pos
	Name   *Lit
	Stmt   *Stmt
}

func (c *CoprocClause) Pos() Pos { return c.Coproc }
func (c *CoprocClause) End() Pos { return c.Stmt.End() }

// LetClause represents a Bash let clause.
//
// This node will only appear in LangBash and LangMirBSDKorn.
type LetClause struct {
	Let   Pos
	Exprs []ArithmExpr
}

func (l *LetClause) Pos() Pos { return l.Let }
func (l *LetClause) End() Pos { return l.Exprs[len(l.Exprs)-1].End() }

func wordLastEnd(ws []*Word) Pos {
	if len(ws) == 0 {
		return Pos{}
	}
	return ws[len(ws)-1].End()
}
