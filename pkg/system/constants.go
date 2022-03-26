package system

import "os"

const (
	Namespace            = "herd-system"
	ConfigName           = "herd-config"
	DefaultUserNamespace = "herd"
)

var (
	RegistryName  = "registry"
	RegistryImage = "registry:2.7.1"
	RegistryPort  = 5000
	BuildkitImage = "moby/buildkit:master"
	BuildKitName  = "buildkitd"
	BuildkitPort  = 8080
	AppInitImage  = ""
)

func UserNamespace() string {
	ns := os.Getenv("NAMESPACE")
	if ns != "" {
		return ns
	}
	return DefaultUserNamespace
}
