package condition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/meta"
	"github.com/acorn-io/baaah/pkg/router"
)

type Conditions interface {
	meta.Object
	Conditions() *map[string]v1.Condition
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
	conditions := c.cond.Conditions()
	if *conditions == nil {
		*conditions = map[string]v1.Condition{}
	}
	(*conditions)[c.name] = cond
	c.resp.Objects(c.cond)
}
