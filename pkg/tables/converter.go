package tables

import (
	"bytes"
	"context"
	"html/template"
	"strings"

	"github.com/Masterminds/sprig"
	"github.com/rancher/wrangler-cli/pkg/table"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Converter struct {
	colDefs  []metav1.TableColumnDefinition
	template *template.Template
}

func MustConverter(tableDef [][]string) *Converter {
	c, err := NewConverter(tableDef)
	if err != nil {
		panic(err)
	}
	return c
}

func NewConverter(tableDef [][]string) (*Converter, error) {
	var colDefs []metav1.TableColumnDefinition

	for _, kv := range tableDef {
		colDefs = append(colDefs, metav1.TableColumnDefinition{
			Name:     kv[0],
			Type:     "string",
			Priority: 0,
		})
	}

	_, valueFormat := table.SimpleFormat(tableDef)

	funcs := sprig.FuncMap()
	for k, v := range localFuncMap {
		funcs[k] = v
	}

	t, err := template.New("").Funcs(funcs).Parse(valueFormat)
	if err != nil {
		return nil, err
	}

	c := Converter{
		colDefs:  colDefs,
		template: t,
	}

	return &c, nil
}

func (c Converter) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	var rows []metav1.TableRow

	appendRow := func(obj runtime.Object) error {
		out := &bytes.Buffer{}
		if err := c.template.Execute(out, object); err != nil {
			return err
		}
		var (
			cells []interface{}
			raw   runtime.Object
		)

		if opt, ok := tableOptions.(*metav1.TableOptions); ok {
			switch opt.IncludeObject {
			case metav1.IncludeObject:
				raw = object
			}
		}

		for _, cell := range strings.Split(out.String(), "\t") {
			cells = append(cells, cell)
		}

		rows = append(rows, metav1.TableRow{
			Cells: cells,
			Object: runtime.RawExtension{
				Object: raw,
			},
		})

		return nil
	}

	if meta.IsListType(object) {
		err := meta.EachListItem(object, appendRow)
		if err != nil {
			return nil, err
		}
	} else if err := appendRow(object); err != nil {
		return nil, err
	}

	return &metav1.Table{
		ColumnDefinitions: c.colDefs,
		Rows:              rows,
	}, nil
}
