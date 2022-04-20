package appdefinition

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
)

func (a *AppDefinition) BuildParams() (*v1.ParamSpec, error) {
	return a.params("params.build")
}

func (a *AppDefinition) DeployParams() (*v1.ParamSpec, error) {
	return a.params("params.deploy")
}

func (a *AppDefinition) params(section string) (*v1.ParamSpec, error) {
	app, err := a.ctx.Value()
	if err != nil {
		return nil, err
	}

	v := app.LookupPath(cue.ParsePath(section))
	sv, err := v.Struct()
	if err != nil {
		return nil, err
	}

	// I have no clue what I'm doing here, just poked around
	// until something worked

	result := &v1.ParamSpec{}
	node := v.Syntax(cue.Docs(true))
	s, ok := node.(*ast.StructLit)
	if !ok {
		return result, nil
	}

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
