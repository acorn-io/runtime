package helper

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func MustReturn[T any](f func() (T, error)) T {
	o, err := f()
	Must(err)
	return o
}
