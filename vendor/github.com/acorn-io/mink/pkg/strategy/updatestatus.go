package strategy

import (
	"k8s.io/apimachinery/pkg/runtime"
)

func NewUpdateStatus(schema *runtime.Scheme, strategy StatusUpdater) *UpdateAdapter {
	return &UpdateAdapter{
		CreateAdapter: NewCreate(schema, strategy),
		strategy:      strategy,
		status:        true,
	}
}
