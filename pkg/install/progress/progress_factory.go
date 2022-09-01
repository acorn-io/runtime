package progress

type Builder interface {
	New(component string) Progress
}

type Progress interface {
	Infof(fmt string, args ...interface{})
	Fail(err error) error
	SuccessWithWarning(fmt string, args ...interface{})
	Success()
}
