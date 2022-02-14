package cue

import (
	"io/fs"
	"io/ioutil"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
)

type Context struct {
	files []File
	fses  []fsEntry
}

type fsEntry struct {
	prepend string
	fs      fs.FS
}

type File struct {
	Name string
	Data []byte
}

func NewContext() *Context {
	return &Context{}
}

func (c Context) clone() *Context {
	return &Context{
		files: c.files,
		fses:  c.fses,
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
	ctx := cuecontext.New()

	// cue needs a dir so we create a unique temporary one for each call.
	// I wish I didn't need this.
	dir, err := ioutil.TempDir("", "herd.cue-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	overrides := map[string]load.Source{}
	if err := AddFiles(overrides, dir, files...); err != nil {
		return nil, err
	}

	for _, entry := range c.fses {
		if err := AddFS(overrides, dir, entry.prepend, entry.fs); err != nil {
			return nil, err
		}
	}

	instances := load.Instances(args, &load.Config{
		Dir:     dir,
		Overlay: overrides,
	})
	if err != nil {
		return nil, err
	}

	values, err := ctx.BuildInstances(instances)
	if err != nil {
		return nil, err
	}

	value := &values[0]
	return value, value.Err()
}

func (c *Context) Transform(path string) (*cue.Value, error) {
	currentValue, err := c.Value()
	if err != nil {
		return nil, err
	}

	transformer, err := c.buildValue([]string{path})
	if err != nil {
		return nil, err
	}

	if transformer.Err() != nil {
		return nil, transformer.Err()
	}

	transformed := transformer.FillPath(cue.ParsePath("in"), currentValue)
	if transformed.Err() != nil {
		return nil, transformed.Err()
	}

	out := transformed.LookupPath(cue.ParsePath("out"))
	return &out, out.Err()
}

func (c *Context) Value() (*cue.Value, error) {
	var args []string
	for _, f := range c.files {
		args = append(args, f.Name)
	}

	return c.buildValue(args, c.files...)
}
