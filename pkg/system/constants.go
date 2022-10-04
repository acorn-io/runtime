package system

import "os"

const (
	Namespace            = "acorn-system"
	ConfigName           = "acorn-config"
	TLSSecretName        = "acorn-tls"
	LEAccountSecretName         = "acorn-le-account"
	DefaultUserNamespace = "acorn"
	DNSSecretName        = "acorn-dns"
)

var (
	RegistryName             = "registry"
	RegistryImage            = "registry:2.7.1"
	NginxImage               = "nginx:1.23.1-alpine"
	RegistryPort             = 5000
	BuildkitImage            = "moby/buildkit:v0.10.3"
	BuildKitName             = "buildkitd"
	ControllerName           = "acorn-controller"
	APIServerName            = "acorn-api"
	BuildkitPort             = 8080
	KlipperLBImage           = "rancher/klipper-lb:v0.3.4"
	IndexURL                 = "https://cdn.acrn.io/ui/latest/index.html"
	ContainerdConfigPathName = "containerd-config-path"
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
