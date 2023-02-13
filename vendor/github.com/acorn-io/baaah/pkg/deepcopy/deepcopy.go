package deepcopy

import (
	"sigs.k8s.io/controller-tools/pkg/deepcopy"
	"sigs.k8s.io/controller-tools/pkg/genall"
)

func Deepcopy(paths ...string) {
	var deepcopy genall.Generator = deepcopy.Generator{}
	for _, dir := range paths {
		runtime, err := genall.Generators{&deepcopy}.ForRoots(dir)
		if err != nil {
			panic(err)
		}
		runtime.OutputRules.Default = genall.OutputToDirectory(dir)
		runtime.Run()
	}
}
