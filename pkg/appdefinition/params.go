package appdefinition

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
)

func (a *AppDefinition) BuildParams() (*v1.ParamSpec, error) {
	app, err := a.ctx.Value()
	if err != nil {
		return nil, err
	}

	v := app.LookupPath(cue.ParsePath("params.build"))
	sv, err := v.Struct()
	if err != nil {
		return nil, err
	}

	node := v.Syntax(cue.Docs(true))
	s := node.(*ast.StructLit)
	result := &v1.ParamSpec{}

	for i, o := range s.Elts {
		f := o.(*ast.Field)
		com := strings.Builder{}
		for _, c := range ast.Comments(o) {
			for _, d := range c.List {
				s := strings.TrimSpace(d.Text)
				s = strings.TrimPrefix(s, "//")
				s = strings.TrimSpace(s)
				com.WriteString(s)
				com.WriteString("\n")
			}
		}
		result.Params = append(result.Params, v1.Param{
			Name:        fmt.Sprint(f.Label),
			Description: strings.TrimSpace(com.String()),
			Schema:      fmt.Sprint(sv.Field(i).Value),
		})
	}

	return result, nil
}
