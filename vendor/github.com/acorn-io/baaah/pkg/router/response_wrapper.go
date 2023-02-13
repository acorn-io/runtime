package router

import (
	"time"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ResponseWrapper struct {
	NoPrune bool
	Delay   time.Duration
	Objs    []kclient.Object
}

func (r *ResponseWrapper) DisablePrune() {
	r.NoPrune = true
}

func (r *ResponseWrapper) RetryAfter(delay time.Duration) {
	r.Delay = delay
}

func (r *ResponseWrapper) Objects(obj ...kclient.Object) {
	r.Objs = append(r.Objs, obj...)
}
