package system

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

	AcornPriorityClass = "system-cluster-critical"
)

var (
	RegistryName                   = "registry"
	RegistryPort                   = 5000
	BuildKitName                   = "buildkitd"
	ControllerName                 = "acorn-controller"
	APIServerName                  = "acorn-api"
	BuildkitPort             int32 = 8080
	ContainerdConfigPathName       = "containerd-config-path"
	DefaultHubAddress              = "acorn.io"
)
