package appdefinition

import (
	"embed"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/parser"
	"github.com/acorn-io/aml"
)

var (
	//go:embed std.cue
	fs  embed.FS
	std aml.StdDef
)

func init() {
	data, err := fs.ReadFile("std.cue")
	if err != nil {
		panic(err)
	}
	stdData, err := parser.ParseFile("std.cue", data)
	if err != nil {
		panic(err)
	}
	functions := map[string]bool{}
	for _, e := range stdData.Decls[1].(*ast.LetClause).Expr.(*ast.StructLit).Elts {
		functions[e.(*ast.Field).Label.(*ast.Ident).Name] = true
	}

	std.Imports = stdData.Imports
	std.Unresolved = stdData.Unresolved
	std.Decls = stdData.Decls
	std.Functions = functions
}

func parseFile(name string, src any) (f *ast.File, err error) {
	return aml.ParseFile(name, src, &std)
}
