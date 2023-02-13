package aml

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/token"
	amlparser "github.com/acorn-io/aml/parser"
	"github.com/acorn-io/baaah/pkg/merr"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/agnivade/levenshtein"
	"github.com/pkg/errors"
)

type needStd struct {
	errs      []error
	needed    bool
	functions map[string]bool
}

func (n *needStd) Needed() bool {
	return n.needed
}

func (n *needStd) Err() error {
	return merr.NewErrors(n.errs...)
}

func bestFunction(name string, functions map[string]bool) []string {
	var (
		match = map[int]string{}
	)
	for _, f := range typed.SortedKeys(functions) {
		d := levenshtein.ComputeDistance(strings.ToLower(name), strings.ToLower(f))
		match[d] = f
	}

	keys := typed.SortedValuesByKey(match)
	if len(keys) < 3 {
		return keys
	}
	return keys[:3]
}

func (n *needStd) Walk(node ast.Node) bool {
	if _, ok := node.(*ast.Package); ok {
		n.errs = append(n.errs, fmt.Errorf("package keyword is not supported"))
	}
	if sel, ok := node.(*ast.SelectorExpr); ok {
		if i, ok := sel.X.(*ast.Ident); ok && i.Name == "std" {
			n.needed = true
			if i, ok := sel.Sel.(*ast.Ident); ok {
				if !n.functions[i.Name] {
					n.errs = append(n.errs, fmt.Errorf("invalid reference to std.%s, closest matches %s %v", i.Name, bestFunction(i.Name, n.functions), sel.Pos()))
				}
			}
		}
	}
	return true
}

type argsOptional struct {
	errs []error
}

func (a *argsOptional) Err() error {
	return merr.NewErrors(a.errs...)
}

func orDefaultList(b *ast.ListLit) ast.Expr {
	return &ast.BinaryExpr{
		X: &ast.UnaryExpr{
			OpPos: b.Pos(),
			Op:    token.MUL,
			X:     b,
		},
		OpPos: b.Pos(),
		Op:    token.OR,
		Y: &ast.ListLit{
			Lbrack: b.Pos(),
			Elts: []ast.Expr{
				&ast.Ellipsis{
					Ellipsis: b.Pos(),
					Type: &ast.Ident{
						NamePos: b.Pos(),
						Name:    "string",
					},
				},
			},
			Rbrack: b.Pos(),
		},
	}
}

func orDefault(b *ast.BasicLit, kind string) ast.Expr {
	return &ast.BinaryExpr{
		X: &ast.UnaryExpr{
			OpPos: b.Pos(),
			Op:    token.MUL,
			X:     b,
		},
		OpPos: b.Pos(),
		Op:    token.OR,
		Y: &ast.Ident{
			NamePos: b.Pos(),
			Name:    kind,
		},
	}
}

func defaultTheLiteral(b *ast.BinaryExpr) ast.Expr {
	if _, ok := b.X.(*ast.BasicLit); ok {
		b.X = &ast.UnaryExpr{
			OpPos: b.X.Pos(),
			Op:    token.MUL,
			X:     b.X,
		}
	} else if b, ok := b.X.(*ast.BinaryExpr); ok {
		defaultTheLiteral(b)
	}
	return b
}

func AllLitStrings(b ast.Expr, allowDefault bool) bool {
	if b, ok := b.(*ast.BasicLit); ok && b.Kind == token.STRING {
		return true
	}
	if b, ok := b.(*ast.BinaryExpr); ok && b.Op == token.OR {
		return AllLitStrings(b.X, allowDefault) && AllLitStrings(b.Y, allowDefault)
	}
	if b, ok := b.(*ast.UnaryExpr); ok && b.Op == token.MUL {
		return AllLitStrings(b.X, allowDefault)
	}
	return false
}

func allStrings(l *ast.ListLit) bool {
	for _, e := range l.Elts {
		l, ok := e.(*ast.BasicLit)
		if !ok {
			return false
		}
		if l.Kind != token.STRING {
			return false
		}
	}
	return true
}

func (a *argsOptional) Walk(node ast.Node) bool {
	f, ok := node.(*ast.Field)
	if !ok {
		return true
	}

	l, ok := f.Label.(*ast.Ident)
	if ok && l.Name == "args" {
		return a.walkFields(f)
	}

	if ok && l.Name == "profiles" {
		s, ok := f.Value.(*ast.StructLit)
		if !ok {
			return false
		}

		for _, e := range s.Elts {
			if _, ok := e.(*ast.Comprehension); ok {
				a.errs = append(a.errs, errors.New("comprehension (if) should not be used inside the args and profiles fields"))
				return false
			}
			f, ok := e.(*ast.Field)
			if !ok {
				return false
			}
			if !a.walkFields(f) {
				return false
			}
		}

	}

	return false
}

func (a *argsOptional) walkFields(f *ast.Field) bool {
	s, ok := f.Value.(*ast.StructLit)
	if !ok {
		return false
	}

	for _, e := range s.Elts {
		if _, ok := e.(*ast.Comprehension); ok {
			a.errs = append(a.errs, errors.New("comprehension (if) should not be used inside the args and profiles fields"))
			return false
		}
		f, ok := e.(*ast.Field)
		if !ok {
			return false
		}
		if b, ok := f.Value.(*ast.BasicLit); ok {
			switch b.Kind {
			case token.STRING:
				f.Value = orDefault(b, "string")
			case token.INT:
				f.Value = orDefault(b, "int")
			case token.FLOAT:
				f.Value = orDefault(b, "float")
			case token.FALSE:
				fallthrough
			case token.TRUE:
				f.Value = orDefault(b, "bool")
			default:
				fmt.Printf("%s", b.Kind)
			}
		} else if l, ok := f.Value.(*ast.ListLit); ok && allStrings(l) {
			f.Value = orDefaultList(l)
		} else if b, ok := f.Value.(*ast.BinaryExpr); ok && AllLitStrings(b, false) {
			f.Value = defaultTheLiteral(b)
		}
	}

	return false
}

type StdDef struct {
	Imports    []*ast.ImportSpec
	Unresolved []*ast.Ident
	Decls      []ast.Decl
	Functions  map[string]bool
}

func ParseFile(name string, src interface{}, std *StdDef) (f *ast.File, err error) {
	file, err := amlparser.ParseFile(name, src, amlparser.ParseComments)
	if err != nil {
		return nil, err
	}
	if len(file.Imports) > 0 {
		return nil, fmt.Errorf("import keyword is not supported")
	}
	args := argsOptional{}
	needStd := needStd{functions: std.Functions}
	for _, decl := range file.Decls {
		ast.Walk(decl, args.Walk, nil)
		ast.Walk(decl, needStd.Walk, nil)
	}

	if needStd.Needed() {
		file.Imports = std.Imports
		file.Decls = append(file.Decls, std.Decls...)
		file.Unresolved = append(file.Unresolved, std.Unresolved...)
	}
	return file, merr.NewErrors(args.Err(), needStd.Err())
}
