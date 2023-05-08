package condition

import (
	"reflect"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Conditions interface {
	kclient.Object
	Conditions() *[]v1.Condition
}

type reflector struct {
	kclient.Object
}

func (r *reflector) Conditions() *[]v1.Condition {
	if c, ok := r.Object.(Conditions); ok {
		return c.Conditions()
	}
	ptr := reflect.ValueOf(r.Object).Elem().FieldByName("Status").FieldByName("Conditions").Addr()
	v := ptr.Interface().(*[]v1.Condition)
	return v
}

func ForName(obj kclient.Object, name string) *Callback {
	return Setter(&reflector{
		Object: obj,
	}, nil, name)
}

func Setter(cond Conditions, resp router.Response, name string) *Callback {
	return &Callback{
		cond: cond,
		name: name,
		resp: resp,
	}
}

type Callback struct {
	name string
	cond Conditions
	resp router.Response
}

func (c *Callback) Success() {
	c.Set(v1.Condition{
		Success: true,
	})
}

func (c *Callback) Unknown(msg string) {
	c.Set(v1.Condition{
		Message:       msg,
		Transitioning: true,
	})
}

func (c *Callback) Error(err error) {
	if err == nil {
		c.Success()
		return
	}
	c.Set(v1.Condition{
		Error:   true,
		Message: err.Error(),
	})
}

func (c *Callback) Set(cond v1.Condition) {
	for i, existing := range *c.cond.Conditions() {
		if existing.Type == c.name {
			(*c.cond.Conditions())[i] = existing.Set(cond, c.cond.GetGeneration())
			return
		}
	}
	*c.cond.Conditions() = append(*c.cond.Conditions(), cond.Init(c.name, c.cond.GetGeneration()))
	if c.resp != nil {
		c.resp.Objects(c.cond)
	}
}
