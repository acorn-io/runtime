package system

const (
	Namespace = "herd-system"
)

var (
	RegistryName  = "registry"
	RegistryImage = "registry:2.7.1"
	RegistryPort  = 5000
	BuildkitImage = "moby/buildkit:master"
	BuildKitName  = "buildkitd"
	BuildkitPort  = 8080
)
