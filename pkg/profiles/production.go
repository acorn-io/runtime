package profiles

import (
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/z"
)

// productionProfile returns a config with production specific
// defaults. Everything else is the same as the default config.
func productionProfile() apiv1.Config {
	conf := defaultProfile()
	conf.AcornDNS = z.Pointer("enabled")
	conf.AllowTrafficFromNamespace = []string{"prometheus-operator"}
	conf.AWSIdentityProviderARN = z.Pointer("{{ .Values.awsIdentityProviderArn | quote }}")
	conf.BuilderPerProject = z.Pointer(true)
	conf.CertManagerIssuer = z.Pointer("letsencrypt-prod")
	conf.IngressControllerNamespace = z.Pointer("traefik")
	conf.LetsEncrypt = z.Pointer("enabled")
	conf.ManageVolumeClasses = z.Pointer(true)
	conf.NetworkPolicies = z.Pointer(true)
	conf.PublishBuilders = z.Pointer(true)
	conf.VolumeSizeDefault = "2Gi"

	// These values are based on internal testing and usage
	// statistics. They are not based on any formal benchmarking.
	conf.RegistryMemory = z.Pointer("128Mi:512Mi")
	conf.RegistryCPU = z.Pointer("200m")
	conf.BuildkitdMemory = z.Pointer("256Mi:10Gi")
	conf.BuildkitdCPU = z.Pointer("800m")
	conf.BuildkitdServiceMemory = z.Pointer("128Mi:256Mi")
	conf.BuildkitdServiceCPU = z.Pointer("200m")
	conf.ControllerMemory = z.Pointer("256Mi")
	conf.ControllerCPU = z.Pointer("100m")
	conf.APIServerMemory = z.Pointer("256Mi")
	conf.APIServerCPU = z.Pointer("100m")

	return conf
}
