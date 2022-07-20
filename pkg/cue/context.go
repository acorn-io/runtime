package cue

import (
	"bytes"
	"fmt"
	"io/fs"
	"sync"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
)

var loadLock sync.Mutex

type ParserFunc func(name string, src interface{}) (*ast.File, error)

type Context struct {
	files     []File
	fses      []fsEntry
	ctx       *cue.Context
	parseFile ParserFunc
}

type fsEntry struct {
	prepend string
	fs      fs.FS
}

type File struct {
	Name   string
	Data   []byte
	Parser ParserFunc
}

func NewContext() *Context {
	return &Context{
		ctx: cuecontext.New(),
	}
}

func (c Context) WithParser(parser ParserFunc) *Context {
	ret := c.clone()
	ret.parseFile = parser
	return ret
}

func (c Context) clone() *Context {
	return &Context{
		files:     c.files,
		fses:      c.fses,
		ctx:       c.ctx,
		parseFile: c.parseFile,
	}
}

func (c Context) WithFile(name string, data []byte) *Context {
	return c.WithFiles(File{
		Name: name,
		Data: data,
	})
}

func (c Context) WithNestedFS(prepend string, fs fs.FS) *Context {
	newC := c.clone()
	newC.fses = append(newC.fses, fsEntry{
		prepend: prepend,
		fs:      fs,
	})
	return newC
}

func (c Context) WithFS(fs ...fs.FS) *Context {
	newC := c.clone()
	for _, v := range fs {
		newC.fses = append(newC.fses, fsEntry{
			fs: v,
		})
	}
	return newC
}

func (c Context) WithFiles(file ...File) *Context {
	newC := c.clone()
	newC.files = append(newC.files, file...)
	return newC
}

func (c *Context) buildValue(args []string, files ...File) (*cue.Value, error) {
	ctx := c.ctx

	overrides := map[string]load.Source{}
	if err := AddFiles(overrides, dir, files...); err != nil {
		return nil, err
	}

	for _, entry := range c.fses {
		if err := AddFS(overrides, dir, entry.prepend, entry.fs); err != nil {
			return nil, err
		}
	}

	// https://github.com/cue-lang/cue/issues/1043
	loadLock.Lock()
	instances := load.Instances(args, &load.Config{
		Dir:       dir,
		Overlay:   overrides,
		ParseFile: c.parseFile,
	})
	loadLock.Unlock()

	values, err := ctx.BuildInstances(instances)
	if err != nil {
		return nil, wrapErr(err)
	}

	value := &values[0]
	return value, wrapErr(value.Err())
}

func (c *Context) Validate(path, typeName string) error {
	currentValue, err := c.Value()
	if err != nil {
		return err
	}

	validation, err := c.buildValue([]string{path})
	if err != nil {
		return err
	}
	schema := validation.LookupPath(cue.ParsePath(typeName))

	newValue := currentValue.Unify(schema)
	if newValue.Err() != nil {
		return wrapErr(newValue.Err())
	}

	return wrapErr(newValue.Validate())
}

func (c *Context) Compile(data []byte) (*cue.Value, error) {
	v := c.ctx.CompileBytes(data)
	return &v, wrapErr(v.Err())
}

func (c *Context) Encode(obj interface{}) (*cue.Value, error) {
	v := c.ctx.Encode(obj)
	return &v, wrapErr(v.Err())
}

func (c *Context) TransformValue(currentValue *cue.Value, path string) (*cue.Value, error) {
	transformer, err := c.buildValue([]string{path})
	if err != nil {
		return nil, err
	}

	if transformer.Err() != nil {
		return nil, wrapErr(transformer.Err())
	}

	transformed := transformer.FillPath(cue.ParsePath("in"), currentValue)
	if transformed.Err() != nil {
		return nil, wrapErr(transformed.Err())
	}

	out := transformed.LookupPath(cue.ParsePath("out"))
	return &out, wrapErr(out.Err())
}

func (c *Context) Transform(path string) (*cue.Value, error) {
	currentValue, err := c.Value()
	if err != nil {
		return nil, err
	}
	return c.TransformValue(currentValue, path)
}

func (c *Context) Value() (*cue.Value, error) {
	var args []string
	for _, f := range c.files {
		args = append(args, f.Name)
	}

	return c.buildValue(args, c.files...)
}

func (c *Context) Decode(v *cue.Value, obj interface{}) error {
	err := v.Decode(obj)
	if err != nil {
		return fmt.Errorf("value %v: %w", v, wrapErr(err))
	}
	return nil
}

func wrapErr(err error) error {
	if err == nil {
		return nil
	}
	return &wrappedErr{Err: err}
}

type wrappedErr struct {
	Err error
}

func (w *wrappedErr) Error() string {
	buf := &bytes.Buffer{}
	errors.Print(buf, w.Err, nil)
	return buf.String()
}

func (w *wrappedErr) Unwrap() error {
	return w.Err
}
