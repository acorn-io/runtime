package appdefinition

import "embed"

var (
	//go:embed app.acorn app-default.acorn
	fs embed.FS
)

const (
	schemaFile  = "app.acorn"
	defaultFile = "app-default.acorn"
)
