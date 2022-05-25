package main

import (
	"fmt"
	"log"
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
	err := doc.GenMarkdownTreeCustom(acorn.New(), "docs/docs/100-Reference/01-command-line", filePrepender, linkHandler)
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
