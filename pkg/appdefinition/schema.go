package appdefinition

import "embed"

var (
	//go:embed acornfile-schema.acorn app-default.acorn
	fs embed.FS
)

const (
	schemaFile  = "acornfile-schema.acorn"
	defaultFile = "app-default.acorn"
)
