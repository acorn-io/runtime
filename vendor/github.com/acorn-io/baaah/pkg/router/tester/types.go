package tester

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/acorn-io/baaah/pkg/randomtoken"
	"github.com/acorn-io/baaah/pkg/uncached"
	"github.com/google/uuid"
	"golang.org/x/exp/maps"
	"k8s.io/apimachinery/pkg/api/errors"
	meta2 "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type Client struct {
	Objects   []kclient.Object
	SchemeObj *runtime.Scheme
	Created   []kclient.Object
	Updated   []kclient.Object
}

func (c Client) objects() []kclient.Object {
	return append(append(c.Objects, c.Created...), c.Updated...)
}

func (c *Client) Get(ctx context.Context, key kclient.ObjectKey, out kclient.Object) error {
	if u, ok := out.(*uncached.Holder); ok {
		out = u.Object
	}
	t := reflect.TypeOf(out)
	var ns string
	if key.Namespace != "" {
		ns = key.Namespace
	}

	// iterate over the slice in reverse because updated objects are at the end. created objects are before them.
	// in the scenario where an object has been created and then updated, or updated twice, we want the last object in
	objs := c.objects()
	for i := len(objs) - 1; i >= 0; i-- {
		obj := objs[i]
		if reflect.TypeOf(obj) != t {
			continue
		}
		if obj.GetName() == key.Name &&
			obj.GetNamespace() == ns {
			copy(out, obj)
			return nil
		}
	}
	return errors.NewNotFound(schema.GroupResource{
		Group:    fmt.Sprintf("Unknown group from test: %T", out),
		Resource: fmt.Sprintf("Unknown resource from test: %T", out),
	}, key.Name)
}

func copy(dest, src kclient.Object) {
	srcCopy := src.DeepCopyObject()
	reflect.Indirect(reflect.ValueOf(dest)).Set(reflect.Indirect(reflect.ValueOf(srcCopy)))
}

func (c *Client) List(ctx context.Context, objList kclient.ObjectList, opts ...kclient.ListOption) error {
	if u, ok := objList.(*uncached.HolderList); ok {
		objList = u.ObjectList
	}

	listOpts := &kclient.ListOptions{}
	for _, opt := range opts {
		opt.ApplyToList(listOpts)
	}

	gvk, err := apiutil.GVKForObject(objList, c.SchemeObj)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(gvk.Kind, "List") {
		return fmt.Errorf("invalid list object %v, Kind must end with List", gvk)
	}
	gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")
	genericObj, err := c.SchemeObj.New(gvk)
	if err != nil {
		return err
	}
	obj := genericObj.(kclient.Object)
	t := reflect.TypeOf(obj)
	var ns string
	if listOpts.Namespace != "" {
		ns = listOpts.Namespace
	}

	// put objects into a map because c.objects() returns both created and updated objects, with updates coming after
	// created. this will ensure the last object in is what is returned
	resultObjs := make(map[string]runtime.Object)
	for _, testObj := range c.objects() {
		if testObj.GetNamespace() != ns {
			continue
		}
		if reflect.TypeOf(testObj) != t {
			continue
		}
		if opts != nil && listOpts.LabelSelector != nil && !listOpts.LabelSelector.Matches(labels.Set(testObj.GetLabels())) {
			continue
		}
		copy(obj, testObj)
		resultObjs[testObj.GetNamespace()+"/"+testObj.GetName()] = testObj
		newObj, err := c.SchemeObj.New(gvk)
		if err != nil {
			return err
		}
		obj = newObj.(kclient.Object)
	}
	return meta2.SetList(objList, maps.Values(resultObjs))
}

func (c *Client) Create(ctx context.Context, obj kclient.Object, opts ...kclient.CreateOption) error {
	obj.SetUID(types.UID(uuid.New().String()))
	if obj.GetName() == "" && obj.GetGenerateName() != "" {
		r, err := randomtoken.Generate()
		if err != nil {
			return err
		}
		obj.SetName(obj.GetGenerateName() + r[:5])
	}
	c.Created = append(c.Created, obj)
	return nil
}

func (c *Client) Update(ctx context.Context, o kclient.Object, opts ...kclient.UpdateOption) error {
	t := reflect.TypeOf(o)

	for _, obj := range c.objects() {
		if reflect.TypeOf(obj) != t {
			continue
		}
		if obj.GetName() == o.GetName() && obj.GetNamespace() == o.GetNamespace() {
			c.Updated = append(c.Updated, o)
			return nil
		}
	}

	return errors.NewNotFound(schema.GroupResource{
		Group:    fmt.Sprintf("Unknown group from test: %T", o),
		Resource: fmt.Sprintf("Unknown resource from test: %T", o),
	}, o.GetName())
}

type Response struct {
	Delay     time.Duration
	Collected []kclient.Object
	Client    *Client
	NoPrune   bool
}

func (r *Response) DisablePrune() {
	r.NoPrune = true
}

func (r *Response) RetryAfter(delay time.Duration) {
	if r.Delay == 0 || delay < r.Delay {
		r.Delay = delay
	}
}

func (r *Response) Objects(obj ...kclient.Object) {
	r.Collected = append(r.Collected, obj...)
}

func (c *Client) Delete(ctx context.Context, obj kclient.Object, opts ...kclient.DeleteOption) error {
	//TODO implement me
	panic("implement me")
}

func (c *Client) Patch(ctx context.Context, obj kclient.Object, patch kclient.Patch, opts ...kclient.PatchOption) error {
	//TODO implement me
	panic("implement me")
}

func (c *Client) DeleteAllOf(ctx context.Context, obj kclient.Object, opts ...kclient.DeleteAllOfOption) error {
	//TODO implement me
	panic("implement me")
}

func (c *Client) Status() kclient.StatusWriter {
	//TODO implement me
	panic("implement me")
}

func (c *Client) Scheme() *runtime.Scheme {
	return c.SchemeObj
}

func (c *Client) RESTMapper() meta2.RESTMapper {
	//TODO implement me
	panic("implement me")
}
