package system

import "os"

const (
	Namespace            = "acorn-system"
	ImagesNamespace      = "acorn-image-system"
	ConfigName           = "acorn-config"
	TLSSecretName        = "acorn-tls"
	LEAccountSecretName  = "acorn-le-account"
	DefaultUserNamespace = "acorn"
	DNSSecretName        = "acorn-dns"

	CustomCABundleSecretName = "cabundle"
	CustomCABundleSecretVolumeName
	CustomCABundleDir      = "/etc/ssl/certs"
	CustomCABundleCertName = "ca-certificates.crt"
)

var (
	RegistryName             = "registry"
	RegistryPort             = 5000
	BuildKitName             = "buildkitd"
	ControllerName           = "acorn-controller"
	APIServerName            = "acorn-api"
	BuildkitPort             = 8080
	ContainerdConfigPathName = "containerd-config-path"
	DefaultHubAddress        = "acorn.io"
)

func UserNamespace() string {
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
