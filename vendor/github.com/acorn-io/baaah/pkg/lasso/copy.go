package lasso

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
)

func CopyInto(dst, src runtime.Object) error {
	src = src.DeepCopyObject()
	dstVal := reflect.ValueOf(dst)
	srcVal := reflect.ValueOf(src)
	if !srcVal.Type().AssignableTo(dstVal.Type()) {
		return fmt.Errorf("type %s not assignable to %s", srcVal.Type(), dstVal.Type())
	}
	reflect.Indirect(dstVal).Set(reflect.Indirect(srcVal))
	return nil
}
