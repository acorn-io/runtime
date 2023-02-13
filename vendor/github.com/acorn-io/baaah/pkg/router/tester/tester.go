package tester

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/yaml"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	yaml2 "sigs.k8s.io/yaml"
)

type Harness struct {
	Scheme         *runtime.Scheme
	Existing       []kclient.Object
	ExpectedOutput []kclient.Object
	ExpectedDelay  time.Duration
}

func genericToTyped(scheme *runtime.Scheme, objs []runtime.Object) ([]kclient.Object, error) {
	result := make([]kclient.Object, 0, len(objs))
	for _, obj := range objs {
		typedObj, err := scheme.New(obj.GetObjectKind().GroupVersionKind())
		if err != nil {
			return nil, err
		}

		bytes, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(bytes, typedObj); err != nil {
			return nil, err
		}
		result = append(result, typedObj.(kclient.Object))
	}
	return result, nil
}

func readFile(scheme *runtime.Scheme, dir, file string) ([]kclient.Object, error) {
	var (
		path = filepath.Join(dir, file)
		objs []kclient.Object
	)

	files, err := os.ReadDir(path + ".d")
	if os.IsNotExist(err) {
	} else if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") {
			continue
		}
		nestedObj, err := readFile(scheme, path+".d", file.Name())
		if err != nil {
			return nil, err
		}
		objs = append(objs, nestedObj...)
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return objs, nil
	} else if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	newObjects, err := yaml.ToObjects(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("unmarshalling %s: %w", path, err)
	}
	typedObjects, err := genericToTyped(scheme, newObjects)
	if err != nil {
		return nil, err
	}
	return append(objs, typedObjects...), nil
}

func FromDir(scheme *runtime.Scheme, path string) (*Harness, kclient.Object, error) {
	input, err := readFile(scheme, path, "input.yaml")
	if err != nil {
		return nil, nil, err
	}

	if len(input) != 1 {
		return nil, nil, fmt.Errorf("%s/%s does not include one input object", path, "input.yaml")
	}

	existing, err := readFile(scheme, path, "existing.yaml")
	if err != nil {
		return nil, nil, err
	}

	expected, err := readFile(scheme, path, "expected.yaml")
	if err != nil {
		return nil, nil, err
	}

	return &Harness{
		Scheme:         scheme,
		Existing:       existing,
		ExpectedOutput: expected,
		ExpectedDelay:  0,
	}, input[0], nil
}

func DefaultTest(t *testing.T, scheme *runtime.Scheme, path string, handler router.HandlerFunc) (result *Response) {
	t.Helper()
	t.Run(path, func(t *testing.T) {
		harness, input, err := FromDir(scheme, path)
		if err != nil {
			t.Fatal(err)
		}
		result, err = harness.Invoke(t, input, handler)
		if err != nil {
			t.Fatal(err)
		}
	})
	return
}

func (b *Harness) InvokeFunc(t *testing.T, input kclient.Object, handler router.HandlerFunc) (*Response, error) {
	t.Helper()
	return b.Invoke(t, input, handler)
}

func NewRequest(t *testing.T, scheme *runtime.Scheme, input kclient.Object, existing ...kclient.Object) router.Request {
	t.Helper()
	gvk, err := apiutil.GVKForObject(input, scheme)
	if err != nil {
		t.Fatal(err)
	}

	return router.Request{
		Client: &Client{
			Objects:   append(existing, input.DeepCopyObject().(kclient.Object)),
			SchemeObj: scheme,
		},
		Object:      input,
		Ctx:         context.TODO(),
		GVK:         gvk,
		Namespace:   input.GetNamespace(),
		Name:        input.GetName(),
		Key:         toKey(input.GetNamespace(), input.GetName()),
		FromTrigger: false,
	}
}

func (b *Harness) Invoke(t *testing.T, input kclient.Object, handler router.Handler) (*Response, error) {
	t.Helper()
	var (
		req  = NewRequest(t, b.Scheme, input, b.Existing...)
		resp = Response{
			Client: req.Client.(*Client),
		}
	)

	err := handler.Handle(req, &resp)
	if err != nil {
		return &resp, err
	}

	expected, err := toObjectMap(b.Scheme, b.ExpectedOutput)
	if err != nil {
		return &resp, err
	}

	collected, err := toObjectMap(b.Scheme, resp.Collected)
	if err != nil {
		return &resp, err
	}

	assert.Equal(t, b.ExpectedDelay, resp.Delay)

	if len(b.ExpectedOutput) == 0 {
		return &resp, nil
	}

	expectedKeys := map[ObjectKey]bool{}
	for k := range expected {
		expectedKeys[k] = true
	}
	collectedKeys := map[ObjectKey]bool{}
	for k := range collected {
		collectedKeys[k] = true
	}

	for key := range collectedKeys {
		assert.Containsf(t, expectedKeys, key, "Unexpected object %s/%s: %v", key.Namespace, key.Name, key.GVK)
	}

	for key := range expectedKeys {
		assert.Containsf(t, collectedKeys, key, "Missing expected object %s/%s: %v", key.Namespace, key.Name, key.GVK)
		if !assert.ObjectsAreEqual(expected[key], collected[key]) {
			left, _ := yaml2.Marshal(expected[key])
			right, _ := yaml2.Marshal(collected[key])

			left = stripLastTransition(left)
			right = stripLastTransition(right)
			assert.Equal(t, string(left), string(right), "object %s/%s (%v) does not match", key.Namespace, key.Name, key.GVK)
		}
	}

	return &resp, nil
}

func stripLastTransition(buf []byte) []byte {
	result := &bytes.Buffer{}
	s := bufio.NewScanner(bytes.NewReader(buf))
	for s.Scan() {
		line := s.Text()
		if strings.Contains(line, "lastTransitionTime:") {
			continue
		}
		result.WriteString(line)
		result.WriteString("\n")
	}
	return result.Bytes()
}

func toObjectMap(scheme *runtime.Scheme, objs []kclient.Object) (map[ObjectKey]kclient.Object, error) {
	result := map[ObjectKey]kclient.Object{}
	for _, o := range objs {
		gvk, err := apiutil.GVKForObject(o, scheme)
		if err != nil {
			return nil, err
		}
		o.GetObjectKind().SetGroupVersionKind(gvk)
		result[ObjectKey{
			GVK:       gvk,
			Namespace: o.GetNamespace(),
			Name:      o.GetName(),
		}] = o
	}
	return result, nil
}

type ObjectKey struct {
	GVK       schema.GroupVersionKind
	Namespace string
	Name      string
}

func toKey(ns, name string) string {
	if ns == "" {
		return name
	}
	return ns + "/" + name
}
