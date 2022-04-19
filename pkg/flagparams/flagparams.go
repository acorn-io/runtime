package flagparams

import (
	"strings"

	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/appdefinition"
	"github.com/ibuildthecloud/herd/pkg/cue"
	"github.com/rancher/wrangler/pkg/data/convert"
	"github.com/spf13/pflag"
)

type Flags struct {
	FlagSet *pflag.FlagSet
	ints    map[string]*int
	strings map[string]*string
}

func New(filename string, param *v1.ParamSpec) *Flags {
	flagToParam := map[string]interface{}{}
	flagSet := pflag.NewFlagSet(filename, pflag.ContinueOnError)
	ints := map[string]*int{}
	stringValues := map[string]*string{}

	for _, param := range param.Params {
		name := strings.ReplaceAll(convert.ToYAMLKey(param.Name), "_", "-")
		flagToParam[name] = param.Name
		if isType(param.Schema, "int") {
			ints[param.Name] = flagSet.Int(name, 0, param.Description)
		} else {
			stringValues[param.Name] = flagSet.String(name, "", param.Description)
		}
	}

	return &Flags{
		ints:    ints,
		strings: stringValues,
		FlagSet: flagSet,
	}
}

func (f *Flags) Parse(args []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}

	if err := f.FlagSet.Parse(args); err != nil {
		return nil, err
	}

	for name, pValue := range f.strings {
		value := *pValue
		if value == "" {
			continue
		} else if strings.HasPrefix(value, "@") {
			fName := value[1:]
			data, err := appdefinition.ReadCUE(fName)
			if err != nil {
				return nil, err
			}
			if !strings.HasSuffix(fName, ".cue") {
				fName += ".cue"
			}
			val, err := cue.NewContext().WithFile(fName, data).Value()
			if err != nil {
				return nil, err
			}
			result[name] = val
		} else {
			result[name] = value
		}
	}

	for name, pValue := range f.ints {
		value := *pValue
		if value == 0 {
			continue
		}
		result[name] = value
	}

	return result, nil
}

func isType(schema, typeName string) bool {
	schema = strings.TrimSpace(schema)
	return schema == typeName || strings.HasSuffix(schema, "| "+typeName)
}
