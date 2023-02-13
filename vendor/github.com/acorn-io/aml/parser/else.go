package parser

import (
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/token"
)

func getOrSetIfClause(decl ast.Decl) *ast.IfClause {
	comp := decl.(*ast.Comprehension)
	if len(comp.Clauses) != 0 {
		if ifClause, ok := comp.Clauses[0].(*ast.IfClause); ok {
			return ifClause
		}
	}

	ifClause := &ast.IfClause{
		If:        comp.Value.Pos(),
		Condition: ast.NewBool(true),
	}
	comp.Clauses = append([]ast.Clause{
		ifClause,
	}, comp.Clauses...)
	return ifClause
}

func and(expr ast.Expr, other ast.Expr) ast.Expr {
	if expr == nil {
		return other
	}
	return &ast.BinaryExpr{
		X:     expr,
		OpPos: expr.Pos(),
		Op:    token.LAND,
		Y:     other,
	}
}

func not(ifCond ast.Expr) ast.Expr {
	return &ast.UnaryExpr{
		OpPos: ifCond.Pos(),
		Op:    token.NOT,
		X:     ifCond,
	}
}

func buildElse(decls []ast.Decl) ast.Decl {
	if len(decls) == 1 {
		return decls[0]
	}
	var oldNotCondition ast.Expr
	for _, decl := range decls {
		ifCond := getOrSetIfClause(decl)
		newNotCondition := not(ifCond.Condition)
		ifCond.Condition = and(oldNotCondition, ifCond.Condition)
		oldNotCondition = and(oldNotCondition, newNotCondition)
	}
	return &ast.EmbedDecl{
		Expr: &ast.StructLit{
			Lbrace: decls[0].Pos(),
			Elts:   decls,
			Rbrace: decls[0].Pos(),
		},
	}
}
