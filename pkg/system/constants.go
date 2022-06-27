package system

import "os"

const (
	Namespace            = "acorn-system"
	ConfigName           = "acorn-config"
	DefaultUserNamespace = "acorn"
	DNSSecretName        = "acorn-dns"
)

var (
	RegistryName   = "registry"
	RegistryImage  = "registry:2.7.1"
	RegistryPort   = 5000
	BuildkitImage  = "moby/buildkit:v0.10.3"
	BuildKitName   = "buildkitd"
	ControllerName = "acorn-controller"
	APIServerName  = "acorn-api"
	BuildkitPort   = 8080
	KlipperLBImage = "rancher/klipper-lb:v0.3.4"
	IndexURL       = "https://cdn.acrn.io/ui/latest/index.html"
)

func UserNamespace() string {
	if os.Getenv("NAMESPACE_ALL") == "true" {
		return ""
	}
	ns := os.Getenv("NAMESPACE")
	if ns != "" {
		return ns
	}
	return DefaultUserNamespace
}

func RequireUserNamespace() string {
	ns := os.Getenv("NAMESPACE")
	if ns != "" {
		return ns
	}
	return DefaultUserNamespace
}
