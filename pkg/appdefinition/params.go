package appdefinition

import (
	"encoding/json"

	"github.com/acorn-io/aml/cli/pkg/flagargs"
	"github.com/acorn-io/aml/pkg/value"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
)

var (
	hiddenProfiles = map[string]struct{}{
		"devMode":     {},
		"autoUpgrade": {},
	}
	hiddenArgs = map[string]struct{}{
		"dev":         {},
		"autoUpgrade": {},
	}
)

func fromNames(names value.Names) (result []v1.Profile) {
	for _, name := range names {
		result = append(result, v1.Profile(name))
	}
	return
}

func dropHiddenProfiles(names value.Names) (result value.Names) {
	for _, profile := range names {
		if _, ok := hiddenProfiles[profile.Name]; ok {
			continue
		}
		result = append(result, profile)
	}
	return
}

func dropHiddenArgs(args []value.ObjectSchemaField) (result []value.ObjectSchemaField) {
	for _, arg := range args {
		if _, ok := hiddenArgs[arg.Key]; ok {
			continue
		}
		result = append(result, arg)
	}
	return
}

func fromFields(in []value.ObjectSchemaField) (result []v1.Field) {
	for _, item := range in {
		result = append(result, fromField(item))
	}
	return
}

func getDefault(v any) string {
	var (
		ts *value.TypeSchema
	)
	ts, ok := v.(*value.TypeSchema)
	if ok {
		v = ts.DefaultValue
	} else {
		v = nil
	}
	if v == nil {
		return ""
	}
	d, _ := json.Marshal(v)
	return string(d)
}

func fromObject(s value.Schema) *v1.Object {
	var (
		in *value.ObjectSchema
		ts *value.TypeSchema
	)
	ts, ok := s.(*value.TypeSchema)
	if ok {
		in = ts.Object
	}

	if in == nil {
		return nil
	}
	return &v1.Object{
		Path:         ts.Path.String(),
		Reference:    ts.Reference,
		Description:  in.Description,
		Fields:       fromFields(in.Fields),
		AllowNewKeys: in.AllowNewKeys,
	}
}

func fromArray(s value.Schema) *v1.Array {
	var (
		in *value.ArraySchema
		ts *value.TypeSchema
	)
	ts, ok := s.(*value.TypeSchema)
	if ok {
		in = ts.Array
	}
	if in == nil {
		return nil
	}
	return &v1.Array{
		Types: fromFieldTypes(in.Valid),
	}
}

func fromConstraints(s value.Schema) (result []v1.Constraint) {
	var (
		in []value.Constraint
		ts *value.TypeSchema
	)
	ts, ok := s.(*value.TypeSchema)
	if ok {
		in = ts.Constraints
	}
	for _, item := range in {
		result = append(result, v1.Constraint{
			Op:    item.Op,
			Right: toString(item.Right),
		})
	}
	return
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	ts, ok := v.(*value.TypeSchema)
	if ok {
		v = fromFieldType(ts)
	}
	s, _ := json.Marshal(v)
	return string(s)
}

func fromAlternates(s value.Schema) (out []v1.FieldType) {
	var (
		in []value.Schema
		ts *value.TypeSchema
	)
	ts, ok := s.(*value.TypeSchema)
	if ok {
		in = ts.Alternates
	}
	return fromFieldTypes(in)
}

func fromFieldTypes(in []value.Schema) (out []v1.FieldType) {
	for _, fieldType := range in {
		out = append(out, *fromFieldType(fieldType))
	}
	return
}

func fromFieldType(in value.Schema) *v1.FieldType {
	if in == nil {
		return nil
	}
	result := &v1.FieldType{
		Kind:        string(in.TargetKind()),
		Object:      fromObject(in),
		Array:       fromArray(in),
		Constraints: fromConstraints(in),
		Default:     getDefault(in),
		Alternates:  fromAlternates(in),
	}
	if result.Default == "" {
		for _, alt := range result.Alternates {
			if alt.Default != "" {
				result.Default = alt.Default
				break
			}
		}
	}

	return result
}

func fromField(in value.ObjectSchemaField) v1.Field {
	return v1.Field{
		Name:        in.Key,
		Description: in.Description,
		Type:        *fromFieldType(in.Schema),
		Match:       in.Match,
		Optional:    in.Optional,
	}
}

type Flags interface {
	Parse(args []string) (map[string]any, []string, error)
}

func (a *AppDefinition) ToFlags(programName, argsFile string, usage func()) (Flags, error) {
	var file value.FuncSchema
	err := a.decode(&file)
	if err != nil {
		return nil, err
	}

	args := flagargs.New(argsFile, programName,
		dropHiddenProfiles(file.ProfileNames),
		dropHiddenArgs(file.Args))
	args.Usage = usage
	return args, nil
}

func (a *AppDefinition) ToParamSpec() (*v1.ParamSpec, error) {
	var file value.FuncSchema
	err := a.decode(&file)
	if err != nil {
		return nil, err
	}
	result := &v1.ParamSpec{
		Args:     fromFields(dropHiddenArgs(file.Args)),
		Profiles: fromNames(dropHiddenProfiles(file.ProfileNames)),
	}

	return result, nil
}
