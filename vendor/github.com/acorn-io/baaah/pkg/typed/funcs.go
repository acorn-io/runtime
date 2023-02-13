package typed

import "reflect"

func New[T any]() T {
	var empty T
	t := reflect.TypeOf(empty)
	return reflect.New(t.Elem()).Interface().(T)
}

func NewAs[T any, R any]() R {
	var empty T
	t := reflect.TypeOf(empty)
	return reflect.New(t.Elem()).Interface().(R)
}
