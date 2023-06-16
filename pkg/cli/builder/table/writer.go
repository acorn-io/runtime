package table

import (
	"io"
	"os"
	"text/tabwriter"
	"text/template"

	"github.com/acorn-io/aml"
	"golang.org/x/exp/maps"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	yaml2 "sigs.k8s.io/yaml"
)

type Writer interface {
	Write(obj any)
	Close() error
	Err() error
	AddFormatFunc(name string, f FormatFunc)
	SetFormat(format string)
	SetValues(values [][]string, quiet bool)
}

type writer struct {
	closed        bool
	HeaderFormat  string
	ValueFormat   string
	err           error
	headerPrinted bool
	Writer        io.Writer
	funcMap       map[string]any
}

type FormatFunc any

func NewWriter(values [][]string, quiet bool, format string) Writer {
	t := &writer{
		funcMap: maps.Clone(FuncMap),
	}

	t.Writer = tabwriter.NewWriter(os.Stdout, 10, 1, 3, ' ', 0)

	t.SetValues(values, quiet)

	t.SetFormat(format)

	return t
}

func (t *writer) SetValues(values [][]string, quiet bool) {
	t.HeaderFormat, t.ValueFormat = SimpleFormat(values)

	if quiet {
		t.HeaderFormat = ""
		t.ValueFormat = "{{.Obj | fullID }}\n"
		for _, row := range values {
			if len(row) > 1 && row[0] == "Name" {
				_, t.ValueFormat = SimpleFormat([][]string{row})
			}
		}
	}
}

func (t *writer) SetFormat(format string) {
	switch customFormat := format; customFormat {
	case "json":
		t.HeaderFormat = ""
		t.ValueFormat = "json"
	case "jsoncompact":
		t.HeaderFormat = ""
		t.ValueFormat = "jsoncompact"
	case "yaml":
		t.HeaderFormat = ""
		t.ValueFormat = "yaml"
	case "aml":
		t.HeaderFormat = ""
		t.ValueFormat = "aml"
	case "wide", "raw", "table":
	default:
		if customFormat != "" {
			t.ValueFormat = customFormat + "\n"
			t.HeaderFormat = ""
		}
	}
}

func (t *writer) AddFormatFunc(name string, f FormatFunc) {
	t.funcMap[name] = f
}

func (t *writer) Err() error {
	return t.Close()
}

func (t *writer) writeHeader() {
	if t.HeaderFormat != "" && !t.headerPrinted {
		t.headerPrinted = true
		t.err = t.printTemplate(t.Writer, t.HeaderFormat, struct{}{})
		if t.err != nil {
			return
		}
	}
}

func (t *writer) Write(obj any) {
	if t.err != nil {
		return
	}

	if obj, ok := obj.(kclient.Object); ok {
		obj.SetManagedFields(nil)
	}

	t.writeHeader()
	if t.err != nil {
		return
	}

	switch t.ValueFormat {
	case "json":
		content, err := FormatJSON(obj)
		t.err = err
		if t.err != nil {
			return
		}
		_, t.err = t.Writer.Write([]byte(content + "\n"))
	case "jsoncompact":
		content, err := FormatJSONCompact(obj)
		t.err = err
		if t.err != nil {
			return
		}
		_, t.err = t.Writer.Write([]byte(content))
	case "yaml":
		content, err := FormatJSON(obj)
		t.err = err
		if t.err != nil {
			return
		}
		converted, err := yaml2.JSONToYAML([]byte(content))
		t.err = err
		if t.err != nil {
			return
		}
		_, t.err = t.Writer.Write([]byte("---\n"))
		if t.err != nil {
			return
		}
		_, t.err = t.Writer.Write(append(converted, []byte("\n")...))
	case "aml":
		content, err := aml.Marshal(obj)
		t.err = err
		if t.err != nil {
			return
		}
		_, t.err = t.Writer.Write([]byte(string(content) + "\n"))
	default:
		t.err = t.printTemplate(t.Writer, t.ValueFormat, obj)
	}
}

func (t *writer) Close() error {
	if t.closed {
		return t.err
	}
	if t.err != nil {
		return t.err
	}

	defer func() {
		t.closed = true
	}()
	t.writeHeader()
	if t.err != nil {
		return t.err
	}
	if _, ok := t.Writer.(*tabwriter.Writer); ok {
		return t.Writer.(*tabwriter.Writer).Flush()
	}
	return nil
}

func (t *writer) printTemplate(out io.Writer, templateContent string, obj any) error {
	tmpl, err := template.New("").Funcs(t.funcMap).Parse(templateContent)
	if err != nil {
		return err
	}

	return tmpl.Execute(out, obj)
}
