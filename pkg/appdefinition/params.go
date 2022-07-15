package appdefinition

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/aml"
)

func (a *AppDefinition) Args() (*v1.ParamSpec, error) {
	return a.addProfiles(a.args("args"))
}

func (a *AppDefinition) addProfiles(paramSpec *v1.ParamSpec, err error) (*v1.ParamSpec, error) {
	if err != nil {
		return nil, err
	}

	profiles, err := a.args("profiles")
	if err != nil {
		return nil, err
	}

	for _, profile := range profiles.Params {
		paramSpec.Profiles = append(paramSpec.Profiles, v1.Profile{
			Name:        profile.Name,
			Description: profile.Description,
		})
	}

	return paramSpec, nil
}

func (a *AppDefinition) args(section string) (*v1.ParamSpec, error) {
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
		if fmt.Sprint(f.Label) == "dev" {
			continue
		}
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
			Type:        getType(sv.Field(i).Value, f.Value),
		})
	}

	return result, nil
}

func getType(v cue.Value, expr ast.Expr) string {
	if _, err := v.String(); err == nil {
		if aml.AllLitStrings(expr, true) {
			return "enum"
		}
		return "string"
	}
	if _, err := v.Bool(); err == nil {
		return "bool"
	}
	if _, err := v.Int(nil); err == nil {
		return "int"
	}
	if _, err := v.Float64(); err == nil {
		return "float"
	}
	if _, err := v.List(); err == nil {
		return "array"
	}
	return "object"
}
