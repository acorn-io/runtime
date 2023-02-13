package types

import (
	"k8s.io/apimachinery/pkg/fields"
)

type Fields interface {
	fields.Fields
	FieldNames() []string
}
