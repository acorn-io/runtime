package system

import "os"

const (
	Namespace            = "acorn-system"
	ConfigName           = "acorn-config"
	TLSSecretName        = "acorn-tls"
	LEAccountSecretName  = "acorn-le-account"
	DefaultUserNamespace = "acorn"
	DNSSecretName        = "acorn-dns"
)

var (
	RegistryName             = "registry"
	RegistryPort             = 5000
	BuildKitName             = "buildkitd"
	ControllerName           = "acorn-controller"
	APIServerName            = "acorn-api"
	BuildkitPort             = 8080
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
