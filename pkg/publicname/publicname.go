package publicname

import (
	"strings"

	"github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/runtime/pkg/labels"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Get(obj kclient.Object) string {
	publicName := obj.GetLabels()[labels.AcornPublicName]
	if publicName == "" {
		return obj.GetName()
	}
	return publicName
}

func ForChild(parent kclient.Object, childName string) string {
	return name.SafeConcatName(Get(parent) + "." + childName)
}

func Split(name string) (string, string) {
	i := strings.LastIndex(name, ".")
	if i == -1 || i == len(name)-1 || strings.HasPrefix(name, ".") {
		return name, ""
	}
	return name[:i], name[i+1:]
}
