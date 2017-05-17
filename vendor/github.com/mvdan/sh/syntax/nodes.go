// Copyright (c) 2016, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package syntax

import "fmt"

// Node represents an AST node.
type Node interface {
	// Pos returns the first character of the node
	Pos() Pos
	// End returns the character immediately after the node
	End() Pos
}

// File is a shell program.
type File struct {
	Name string

	Stmts    []*Stmt
	Comments []*Comment

	lines []Pos
}

// Pos is the internal representation of a position within a source
// file.
type Pos uint32

// IsValid reports whether the position is valid. All positions in nodes
// returned by Parse are valid.
func (p Pos) IsValid() bool { return p > 0 }

const maxPos = Pos(^uint32(0))

// Position describes a position within a source file including the line
// and column location. A Position is valid if the line number is > 0.
type Position struct {
	Filename string // if any
	Offset   int    // byte offset, starting at 0
	Line     int    // line number, starting at 1
	Column   int    // column number, starting at 1 (in bytes)
}

// IsValid reports whether the position is valid. All positions in nodes
// returned by Parse are valid.
func (p Position) IsValid() bool { return p.Line > 0 }

// String returns the position in the "file:line:column" form, or
// "line:column" if there is no filename available.
func (p Position) String() string {
	prefix := ""
	if p.Filename != "" {
		prefix = p.Filename + ":"
	}
	return fmt.Sprintf("%s%d:%d", prefix, p.Line, p.Column)
}

func (f *File) Pos() Pos {
	if len(f.Stmts) == 0 {
		return 0
	}
	return f.Stmts[0].Pos()
}

func (f *File) End() Pos {
	if len(f.Stmts) == 0 {
		return 0
	}
	return f.Stmts[len(f.Stmts)-1].End()
}

func (f *File) Position(p Pos) (pos Position) {
	pos.Filename = f.Name
	pos.Offset = int(p) - 1
	if i := searchPos(f.lines, p); i >= 0 {
		pos.Line, pos.Column = i+1, int(p-f.lines[i])
	}
	return
}

func searchPos(a []Pos, x Pos) int {
	i, j := 0, len(a)
	for i < j {
		h := i + (j-i)/2
		if a[h] <= x {
			i = h + 1
		} else {
			j = h
		}
	}
	return i - 1
}

func posMax(p1, p2 Pos) Pos {
	if p2 > p1 {
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
func (c *Comment) End() Pos { return c.Hash + Pos(len(c.Text)) }

// Stmt represents a statement, otherwise known as a compound command.
// It is compromised of a command and other components that may come
// before or after it.
type Stmt struct {
	Cmd        Command
	Position   Pos
	Semicolon  Pos
	Negated    bool
	Background bool
	Assigns    []*Assign
	Redirs     []*Redirect
}

func (s *Stmt) Pos() Pos { return s.Position }
func (s *Stmt) End() Pos {
	if s.Semicolon.IsValid() {
		return s.Semicolon + 1
	}
	end := s.Position
	if s.Negated {
		end++
	}
	if s.Cmd != nil {
		end = s.Cmd.End()
	}
	if len(s.Assigns) > 0 {
		end = posMax(end, s.Assigns[len(s.Assigns)-1].End())
	}
	if len(s.Redirs) > 0 {
		end = posMax(end, s.Redirs[len(s.Redirs)-1].End())
	}
	return end
}

// Command represents all nodes that are simple commands, which are
// directly placed in a Stmt.
//
// These are *CallExpr, *IfClause, *WhileClause, *ForClause,
// *CaseClause, *Block, *Subshell, *BinaryCmd, *FuncDecl, *ArithmCmd,
// *TestClause, *DeclClause, *LetClause, and *CoprocClause.
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
func (*CoprocClause) commandNode() {}

// Assign represents an assignment to a variable.
type Assign struct {
	Append bool
	Name   *Lit
	Index  ArithmExpr
	Value  *Word
	Array  *ArrayExpr
}

func (a *Assign) Pos() Pos {
	if a.Name != nil {
		return a.Name.Pos()
	}
	return a.Value.Pos()
}

func (a *Assign) End() Pos {
	if a.Value != nil {
		return a.Value.End()
	}
	if a.Array != nil {
		return a.Array.End()
	}
	if a.Index != nil {
		return a.Index.End() + 2
	}
	return a.Name.End() + 1
}

// Redirect represents an input/output redirection.
type Redirect struct {
	OpPos      Pos
	Op         RedirOperator
	N          *Lit
	Word, Hdoc *Word
}

func (r *Redirect) Pos() Pos {
	if r.N != nil {
		return r.N.Pos()
	}
	return r.OpPos
}
func (r *Redirect) End() Pos { return r.Word.End() }

// CallExpr represents a command execution or function call.
type CallExpr struct {
	Args []*Word
}

func (c *CallExpr) Pos() Pos { return c.Args[0].Pos() }
func (c *CallExpr) End() Pos { return c.Args[len(c.Args)-1].End() }

// Subshell represents a series of commands that should be executed in a
// nested shell environment.
type Subshell struct {
	Lparen, Rparen Pos
	Stmts          []*Stmt
}

func (s *Subshell) Pos() Pos { return s.Lparen }
func (s *Subshell) End() Pos { return s.Rparen + 1 }

// Block represents a series of commands that should be executed in a
// nested scope.
type Block struct {
	Lbrace, Rbrace Pos
	Stmts          []*Stmt
}

func (b *Block) Pos() Pos { return b.Rbrace }
func (b *Block) End() Pos { return b.Rbrace + 1 }

// IfClause represents an if statement.
type IfClause struct {
	If, Then, Else, Fi Pos
	CondStmts          []*Stmt
	ThenStmts          []*Stmt
	Elifs              []*Elif
	ElseStmts          []*Stmt
}

func (c *IfClause) Pos() Pos { return c.If }
func (c *IfClause) End() Pos { return c.Fi + 2 }

// Elif represents an "else if" case in an if clause.
type Elif struct {
	Elif, Then Pos
	CondStmts  []*Stmt
	ThenStmts  []*Stmt
}

// WhileClause represents a while or an until clause.
type WhileClause struct {
	While, Do, Done Pos
	Until           bool
	CondStmts       []*Stmt
	DoStmts         []*Stmt
}

func (w *WhileClause) Pos() Pos { return w.While }
func (w *WhileClause) End() Pos { return w.Done + 4 }

// ForClause represents a for clause.
type ForClause struct {
	For, Do, Done Pos
	Loop          Loop
	DoStmts       []*Stmt
}

func (f *ForClause) Pos() Pos { return f.For }
func (f *ForClause) End() Pos { return f.Done + 4 }

// Loop holds either *WordIter or *CStyleLoop.
type Loop interface {
	Node
	loopNode()
}

func (*WordIter) loopNode()   {}
func (*CStyleLoop) loopNode() {}

// WordIter represents the iteration of a variable over a series of
// words in a for clause.
type WordIter struct {
	Name *Lit
	List []*Word
}

func (w *WordIter) Pos() Pos { return w.Name.Pos() }
func (w *WordIter) End() Pos { return posMax(w.Name.End(), wordLastEnd(w.List)) }

// CStyleLoop represents the behaviour of a for clause similar to the C
// language.
//
// This node will never appear when in PosixConformant mode.
type CStyleLoop struct {
	Lparen, Rparen   Pos
	Init, Cond, Post ArithmExpr
}

func (c *CStyleLoop) Pos() Pos { return c.Lparen }
func (c *CStyleLoop) End() Pos { return c.Rparen + 2 }

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
	Position  Pos
	BashStyle bool
	Name      *Lit
	Body      *Stmt
}

func (f *FuncDecl) Pos() Pos { return f.Position }
func (f *FuncDecl) End() Pos { return f.Body.End() }

// Word represents a non-empty list of nodes that are contiguous to each
// other. The word is delimeted by word boundaries.
type Word struct {
	Parts []WordPart
}

func (w *Word) Pos() Pos { return w.Parts[0].Pos() }
func (w *Word) End() Pos { return w.Parts[len(w.Parts)-1].End() }

// WordPart represents all nodes that can form a word.
//
// These are *Lit, *SglQuoted, *DblQuoted, *ParamExp, *CmdSubst,
// *ArithmExp, *ProcSubst, and *ExtGlob.
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

// Lit represents an unquoted string consisting of characters that were
// not tokenized.
type Lit struct {
	ValuePos, ValueEnd Pos
	Value              string
}

func (l *Lit) Pos() Pos { return l.ValuePos }
func (l *Lit) End() Pos { return l.ValueEnd }

// SglQuoted represents a string within single quotes.
type SglQuoted struct {
	Position Pos
	Dollar   bool
	Value    string
}

func (q *SglQuoted) Pos() Pos { return q.Position }
func (q *SglQuoted) End() Pos {
	end := q.Position + 2 + Pos(len(q.Value))
	if q.Dollar {
		end++
	}
	return end
}

// DblQuoted represents a list of nodes within double quotes.
type DblQuoted struct {
	Position Pos
	Dollar   bool
	Parts    []WordPart
}

func (q *DblQuoted) Pos() Pos { return q.Position }
func (q *DblQuoted) End() Pos {
	if len(q.Parts) == 0 {
		if q.Dollar {
			return q.Position + 3
		}
		return q.Position + 2
	}
	return q.Parts[len(q.Parts)-1].End() + 1
}

// CmdSubst represents a command substitution.
type CmdSubst struct {
	Left, Right Pos
	Stmts       []*Stmt
}

func (c *CmdSubst) Pos() Pos { return c.Left }
func (c *CmdSubst) End() Pos { return c.Right + 1 }

// ParamExp represents a parameter expansion.
type ParamExp struct {
	Dollar, Rbrace Pos
	Short          bool
	Indirect       bool
	Length         bool
	Param          *Lit
	Index          ArithmExpr
	Slice          *Slice
	Repl           *Replace
	Exp            *Expansion
}

func (p *ParamExp) Pos() Pos { return p.Dollar }
func (p *ParamExp) End() Pos {
	if !p.Short {
		return p.Rbrace + 1
	}
	return p.Param.End()
}

func (p *ParamExp) nakedIndex() bool { return p.Short && p.Index != nil }

// Slice represents character slicing inside a ParamExp.
//
// This node will never appear when in PosixConformant mode.
type Slice struct {
	Offset, Length ArithmExpr
}

// Replace represents a search and replace inside a ParamExp.
type Replace struct {
	All        bool
	Orig, With *Word
}

// Expansion represents string manipulation in a ParamExp other than
// those covered by Replace.
type Expansion struct {
	Op   ParExpOperator
	Word *Word
}

// ArithmExp represents an arithmetic expansion.
type ArithmExp struct {
	Left, Right Pos
	Bracket     bool
	X           ArithmExpr
}

func (a *ArithmExp) Pos() Pos { return a.Left }
func (a *ArithmExp) End() Pos {
	if a.Bracket {
		return a.Right + 1
	}
	return a.Right + 2
}

// ArithmCmd represents an arithmetic command.
//
// This node will never appear when in PosixConformant mode.
type ArithmCmd struct {
	Left, Right Pos
	X           ArithmExpr
}

func (a *ArithmCmd) Pos() Pos { return a.Left }
func (a *ArithmCmd) End() Pos { return a.Right + 2 }

// ArithmExpr represents all nodes that form arithmetic expressions.
//
// These are *BinaryArithm, *UnaryArithm, *ParenArithm, *Lit, and
// *ParamExp.
type ArithmExpr interface {
	Node
	arithmExprNode()
}

func (*BinaryArithm) arithmExprNode() {}
func (*UnaryArithm) arithmExprNode()  {}
func (*ParenArithm) arithmExprNode()  {}
func (*Lit) arithmExprNode()          {}
func (*ParamExp) arithmExprNode()     {}

// BinaryArithm represents a binary expression between two arithmetic
// expression.
//
// If Op is any assign operator, X will be a *Lit whose value is a valid
// name.
//
// Ternary operators like "a ? b : c" are fit into this structure. Thus,
// if Op == Quest, Y will be a *BinaryArithm with Op == Colon. Op can
// only be Colon in that scenario.
type BinaryArithm struct {
	OpPos Pos
	Op    BinAritOperator
	X, Y  ArithmExpr
}

func (b *BinaryArithm) Pos() Pos { return b.X.Pos() }
func (b *BinaryArithm) End() Pos { return b.Y.End() }

// UnaryArithm represents an unary expression over a node, either before
// or after it.
//
// If Op is Inc or Dec, X will be a *Lit whose value is a valid name.
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
		return u.OpPos + 2
	}
	return u.X.End()
}

// ParenArithm represents an expression within parentheses inside an
// ArithmExp.
type ParenArithm struct {
	Lparen, Rparen Pos
	X              ArithmExpr
}

func (p *ParenArithm) Pos() Pos { return p.Lparen }
func (p *ParenArithm) End() Pos { return p.Rparen + 1 }

// CaseClause represents a case (switch) clause.
type CaseClause struct {
	Case, Esac Pos
	Word       *Word
	List       []*PatternList
}

func (c *CaseClause) Pos() Pos { return c.Case }
func (c *CaseClause) End() Pos { return c.Esac + 4 }

// PatternList represents a pattern list (case) within a CaseClause.
type PatternList struct {
	Op       CaseOperator
	OpPos    Pos
	Patterns []*Word
	Stmts    []*Stmt
}

// TestClause represents a Bash extended test clause.
//
// This node will never appear when in PosixConformant mode.
type TestClause struct {
	Left, Right Pos
	X           TestExpr
}

func (t *TestClause) Pos() Pos { return t.Left }
func (t *TestClause) End() Pos { return t.Right + 2 }

// TestExpr represents all nodes that form arithmetic expressions.
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

// BinaryTest represents a binary expression between two arithmetic
// expression.
type BinaryTest struct {
	OpPos Pos
	Op    BinTestOperator
	X, Y  TestExpr
}

func (b *BinaryTest) Pos() Pos { return b.X.Pos() }
func (b *BinaryTest) End() Pos { return b.Y.End() }

// UnaryTest represents an unary expression over a node, either before
// or after it.
type UnaryTest struct {
	OpPos Pos
	Op    UnTestOperator
	X     TestExpr
}

func (u *UnaryTest) Pos() Pos { return u.OpPos }
func (u *UnaryTest) End() Pos { return u.X.End() }

// ParenTest represents an expression within parentheses inside an
// TestExp.
type ParenTest struct {
	Lparen, Rparen Pos
	X              TestExpr
}

func (p *ParenTest) Pos() Pos { return p.Lparen }
func (p *ParenTest) End() Pos { return p.Rparen + 1 }

// DeclClause represents a Bash declare clause.
//
// This node will never appear when in PosixConformant mode.
type DeclClause struct {
	Position Pos
	Variant  string
	Opts     []*Word
	Assigns  []*Assign
}

func (d *DeclClause) Pos() Pos { return d.Position }
func (d *DeclClause) End() Pos {
	if len(d.Assigns) > 0 {
		return d.Assigns[len(d.Assigns)-1].End()
	}
	return wordLastEnd(d.Opts)
}

// ArrayExpr represents a Bash array expression.
//
// This node will never appear when in PosixConformant mode.
type ArrayExpr struct {
	Lparen, Rparen Pos
	List           []*Word
}

func (a *ArrayExpr) Pos() Pos { return a.Lparen }
func (a *ArrayExpr) End() Pos { return a.Rparen + 1 }

// ExtGlob represents a Bash extended globbing expression. Note that
// these are parsed independently of whether shopt has been called or
// not.
//
// This node will never appear when in PosixConformant mode.
type ExtGlob struct {
	OpPos   Pos
	Op      GlobOperator
	Pattern *Lit
}

func (e *ExtGlob) Pos() Pos { return e.OpPos }
func (e *ExtGlob) End() Pos { return e.Pattern.End() + 1 }

// ProcSubst represents a Bash process substitution.
//
// This node will never appear when in PosixConformant mode.
type ProcSubst struct {
	OpPos, Rparen Pos
	Op            ProcOperator
	Stmts         []*Stmt
}

func (s *ProcSubst) Pos() Pos { return s.OpPos }
func (s *ProcSubst) End() Pos { return s.Rparen + 1 }

// CoprocClause represents a Bash coproc clause.
//
// This node will never appear when in PosixConformant mode.
type CoprocClause struct {
	Coproc Pos
	Name   *Lit
	Stmt   *Stmt
}

func (c *CoprocClause) Pos() Pos { return c.Coproc }
func (c *CoprocClause) End() Pos { return c.Stmt.End() }

// LetClause represents a Bash let clause.
//
// This node will never appear when in PosixConformant mode.
type LetClause struct {
	Let   Pos
	Exprs []ArithmExpr
}

func (l *LetClause) Pos() Pos { return l.Let }
func (l *LetClause) End() Pos { return l.Exprs[len(l.Exprs)-1].End() }

func wordLastEnd(ws []*Word) Pos {
	if len(ws) == 0 {
		return 0
	}
	return ws[len(ws)-1].End()
}
