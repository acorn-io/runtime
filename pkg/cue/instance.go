package cue

import (
	"io/fs"
	"path/filepath"

	"cuelang.org/go/cue/load"
)

func AddFS(target map[string]load.Source, cwd, prependPath string, files fs.FS) error {
	return fs.WalkDir(files, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		data, err := fs.ReadFile(files, path)
		if err != nil {
			return err
		}

		target[filepath.Join(cwd, prependPath, path)] = load.FromBytes(data)
		return nil
	})
}

func AddFiles(target map[string]load.Source, cwd string, files ...File) error {
	for _, f := range files {
		if f.Parser == nil {
			target[filepath.Join(cwd, f.Name)] = load.FromBytes(f.Data)
		} else {
			ast, err := f.Parser(f.Name, f.Data)
			if err != nil {
				return err
			}
			target[filepath.Join(cwd, f.Name)] = load.FromFile(ast)
		}
	}

	return nil
}
