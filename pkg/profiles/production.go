package profiles

import (
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/z"
)

// productionProfile returns a config with production specific
// defaults. Everything else is the same as the default config.
func productionProfile() apiv1.Config {
	conf := defaultProfile()
	conf.AcornDNS = z.P("enabled")
	conf.AllowTrafficFromNamespace = []string{"prometheus-operator"}
	conf.AWSIdentityProviderARN = z.P("{{ .Values.awsIdentityProviderArn | quote }}")
	conf.BuilderPerProject = z.P(true)
	conf.CertManagerIssuer = z.P("letsencrypt-prod")
	conf.IngressControllerNamespace = z.P("traefik")
	conf.LetsEncrypt = z.P("enabled")
	conf.ManageVolumeClasses = z.P(true)
	conf.NetworkPolicies = z.P(true)
	conf.PublishBuilders = z.P(true)

	// These values are based on internal testing and usage
	// statistics. They are not based on any formal benchmarking.
	conf.RegistryMemory = z.P("128Mi:512Mi")
	conf.RegistryCPU = z.P("200m")
	conf.BuildkitdMemory = z.P("256Mi:10Gi")
	conf.BuildkitdCPU = z.P("800m")
	conf.BuildkitdServiceMemory = z.P("128Mi:256Mi")
	conf.BuildkitdServiceCPU = z.P("200m")
	conf.ControllerMemory = z.P("256Mi")
	conf.ControllerCPU = z.P("100m")
	conf.APIServerMemory = z.P("256Mi")
	conf.APIServerCPU = z.P("100m")

	return conf
}
