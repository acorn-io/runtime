package flagparams

import (
	"os"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/cue"
	"github.com/rancher/wrangler/pkg/data/convert"
	"github.com/spf13/pflag"
)

type Flags struct {
	FlagSet       *pflag.FlagSet
	paramToFlag   map[string]string
	ints          map[string]*int
	strings       map[string]*string
	bools         map[string]*bool
	complexValues map[string]*string
	Usage         func()
}

func New(filename string, param *v1.ParamSpec) *Flags {
	paramToFlag := map[string]string{}
	flagSet := pflag.NewFlagSet(filename, pflag.ContinueOnError)
	ints := map[string]*int{}
	stringValues := map[string]*string{}
	bools := map[string]*bool{}
	complexValues := map[string]*string{}

	for _, param := range param.Params {
		name := strings.ReplaceAll(convert.ToYAMLKey(param.Name), "_", "-")
		paramToFlag[param.Name] = name
		if isType(param.Schema, "int") || isType(param.Schema, "uint") {
			ints[param.Name] = flagSet.Int(name, 0, param.Description)
		} else if isType(param.Schema, "string") {
			stringValues[param.Name] = flagSet.String(name, "", param.Description)
		} else if isType(param.Schema, "bool") {
			bools[param.Name] = flagSet.Bool(name, false, param.Description)
		} else {
			complexValues[param.Name] = flagSet.String(name, "", param.Description)
		}
	}

	return &Flags{
		ints:          ints,
		strings:       stringValues,
		bools:         bools,
		complexValues: complexValues,
		paramToFlag:   paramToFlag,
		FlagSet:       flagSet,
	}
}

func (f *Flags) Parse(args []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}

	if f.Usage != nil {
		f.FlagSet.Usage = func() {
			f.Usage()
			f.FlagSet.PrintDefaults()
		}
	}

	if err := f.FlagSet.Parse(args); err != nil {
		return nil, err
	}

	for name, pValue := range f.complexValues {
		value := *pValue
		if value == "" {
			if !f.flagChanged(name) {
				continue
			}
			result[name] = value
		} else if strings.HasPrefix(value, "@") {
			fName := value[1:]
			data, err := cue.ReadCUE(fName)
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

	for name, pValue := range f.strings {
		value := *pValue
		if value == "" {
			if !f.flagChanged(name) {
				continue
			}
			result[name] = value
		} else if strings.HasPrefix(value, "@") {
			fName := value[1:]
			data, err := os.ReadFile(fName)
			if err != nil {
				return nil, err
			}
			result[name] = string(data)
		} else {
			result[name] = value
		}
	}

	for name, pValue := range f.ints {
		value := *pValue
		if value == 0 {
			if !f.flagChanged(name) {
				continue
			}
		}
		result[name] = value
	}

	for name, pValue := range f.bools {
		value := *pValue
		if !value {
			if !f.flagChanged(name) {
				continue
			}
		}
		result[name] = value
	}

	return result, nil
}
func (f *Flags) flagChanged(name string) bool {
	if fName, ok := f.paramToFlag[name]; ok {
		return f.FlagSet.Lookup(fName).Changed
	}
	return false
}

func isType(schema, typeName string) bool {
	schema = strings.TrimSpace(schema)
	if schema == typeName || strings.HasSuffix(schema, "| "+typeName) {
		return true
	}
	for _, w := range strings.Split(schema, " ") {
		if w == typeName {
			return true
		}
	}
	return false
}
