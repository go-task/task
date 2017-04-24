// Copyright (c) 2016, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package syntax

import "fmt"

func walkStmts(stmts []*Stmt, f func(Node) bool) {
	for _, s := range stmts {
		Walk(s, f)
	}
}

func walkWords(words []*Word, f func(Node) bool) {
	for _, w := range words {
		Walk(w, f)
	}
}

// Walk traverses an AST in depth-first order: It starts by calling
// f(node); node must not be nil. If f returns true, Walk invokes f
// recursively for each of the non-nil children of node, followed by
// f(nil).
func Walk(node Node, f func(Node) bool) {
	if !f(node) {
		return
	}

	switch x := node.(type) {
	case *File:
		walkStmts(x.Stmts, f)
	case *Stmt:
		if x.Cmd != nil {
			Walk(x.Cmd, f)
		}
		for _, a := range x.Assigns {
			Walk(a, f)
		}
		for _, r := range x.Redirs {
			Walk(r, f)
		}
	case *Assign:
		if x.Name != nil {
			Walk(x.Name, f)
		}
		if x.Value != nil {
			Walk(x.Value, f)
		}
	case *Redirect:
		if x.N != nil {
			Walk(x.N, f)
		}
		Walk(x.Word, f)
		if x.Hdoc != nil {
			Walk(x.Hdoc, f)
		}
	case *CallExpr:
		walkWords(x.Args, f)
	case *Subshell:
		walkStmts(x.Stmts, f)
	case *Block:
		walkStmts(x.Stmts, f)
	case *IfClause:
		walkStmts(x.CondStmts, f)
		walkStmts(x.ThenStmts, f)
		for _, elif := range x.Elifs {
			walkStmts(elif.CondStmts, f)
			walkStmts(elif.ThenStmts, f)
		}
		walkStmts(x.ElseStmts, f)
	case *WhileClause:
		walkStmts(x.CondStmts, f)
		walkStmts(x.DoStmts, f)
	case *UntilClause:
		walkStmts(x.CondStmts, f)
		walkStmts(x.DoStmts, f)
	case *ForClause:
		Walk(x.Loop, f)
		walkStmts(x.DoStmts, f)
	case *WordIter:
		Walk(x.Name, f)
		walkWords(x.List, f)
	case *CStyleLoop:
		if x.Init != nil {
			Walk(x.Init, f)
		}
		if x.Cond != nil {
			Walk(x.Cond, f)
		}
		if x.Post != nil {
			Walk(x.Post, f)
		}
	case *BinaryCmd:
		Walk(x.X, f)
		Walk(x.Y, f)
	case *FuncDecl:
		Walk(x.Name, f)
		Walk(x.Body, f)
	case *Word:
		for _, wp := range x.Parts {
			Walk(wp, f)
		}
	case *Lit:
	case *SglQuoted:
	case *DblQuoted:
		for _, wp := range x.Parts {
			Walk(wp, f)
		}
	case *CmdSubst:
		walkStmts(x.Stmts, f)
	case *ParamExp:
		if x.Param != nil {
			Walk(x.Param, f)
		}
		if x.Ind != nil {
			Walk(x.Ind.Expr, f)
		}
		if x.Repl != nil {
			Walk(x.Repl.Orig, f)
			Walk(x.Repl.With, f)
		}
		if x.Exp != nil {
			Walk(x.Exp.Word, f)
		}
	case *ArithmExp:
		if x.X != nil {
			Walk(x.X, f)
		}
	case *ArithmCmd:
		if x.X != nil {
			Walk(x.X, f)
		}
	case *BinaryArithm:
		Walk(x.X, f)
		Walk(x.Y, f)
	case *BinaryTest:
		Walk(x.X, f)
		Walk(x.Y, f)
	case *UnaryArithm:
		Walk(x.X, f)
	case *UnaryTest:
		Walk(x.X, f)
	case *ParenArithm:
		Walk(x.X, f)
	case *ParenTest:
		Walk(x.X, f)
	case *CaseClause:
		Walk(x.Word, f)
		for _, pl := range x.List {
			walkWords(pl.Patterns, f)
			walkStmts(pl.Stmts, f)
		}
	case *TestClause:
		Walk(x.X, f)
	case *DeclClause:
		walkWords(x.Opts, f)
		for _, a := range x.Assigns {
			Walk(a, f)
		}
	case *ArrayExpr:
		walkWords(x.List, f)
	case *ExtGlob:
		Walk(x.Pattern, f)
	case *ProcSubst:
		walkStmts(x.Stmts, f)
	case *EvalClause:
		if x.Stmt != nil {
			Walk(x.Stmt, f)
		}
	case *CoprocClause:
		if x.Name != nil {
			Walk(x.Name, f)
		}
		Walk(x.Stmt, f)
	case *LetClause:
		for _, expr := range x.Exprs {
			Walk(expr, f)
		}
	default:
		panic(fmt.Sprintf("syntax.Walk: unexpected node type %T", x))
	}

	f(nil)
}
