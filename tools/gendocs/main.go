package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	acorn "github.com/acorn-io/acorn/pkg/cli"
	"github.com/spf13/cobra/doc"
)

const fmTemplate = `---
title: "%s"
---
`

func main() {
	cmd := acorn.New()
	cmd.DisableAutoGenTag = true

	files, err := filepath.Glob("docs/docs/100-Reference/01-command-line/acorn_*")
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			log.Fatal(err)
		}
	}

	err = doc.GenMarkdownTreeCustom(cmd, "docs/docs/100-Reference/01-command-line", filePrepender, linkHandler)
	if err != nil {
		log.Fatal(err)
	}
}

func filePrepender(filename string) string {
	name := filepath.Base(filename)
	base := strings.TrimSuffix(name, path.Ext(name))
	return fmt.Sprintf(fmTemplate, strings.Replace(base, "_", " ", -1))
}

func linkHandler(name string) string {
	return name
}
