package condition

import (
	"github.com/ibuildthecloud/baaah/pkg/meta"
	"github.com/ibuildthecloud/baaah/pkg/router"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
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

func (c *Callback) Unknown() {
	c.Set(v1.Condition{})
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
