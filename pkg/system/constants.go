package system

const (
	Namespace            = "acorn-system"
	ImagesNamespace      = "acorn-image-system"
	ConfigName           = "acorn-config"
	DevConfigName        = "acorn-config-dev"
	TLSSecretName        = "acorn-tls"
	LEAccountSecretName  = "acorn-le-account"
	DefaultUserNamespace = "acorn"
	DNSSecretName        = "acorn-dns"
	DNSIngressName       = "acorn-dns-ingress"
	DNSServiceName       = "acorn-dns-service"

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
	DefaultManagerAddress          = "acorn.io"
)
