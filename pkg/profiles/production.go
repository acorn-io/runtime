package profiles

import apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"

// productionProfile returns a config with production specific
// defaults. Everything else is the same as the default config.
func productionProfile() apiv1.Config {
	conf := defaultProfile()
	conf.AcornDNS = ptr("enabled")
	conf.AllowTrafficFromNamespace = []string{"prometheus-operator"}
	conf.AWSIdentityProviderARN = ptr("{{ .Values.awsIdentityProviderArn | quote }}")
	conf.BuilderPerProject = ptr(true)
	conf.CertManagerIssuer = ptr("letsencrypt-prod")
	conf.IngressControllerNamespace = ptr("traefik")
	conf.LetsEncrypt = ptr("enabled")
	conf.ManageVolumeClasses = ptr(true)
	conf.NetworkPolicies = ptr(true)
	conf.PublishBuilders = ptr(true)

	// These values are based on internal testing and usage
	// statistics. They are not based on any formal benchmarking.
	conf.RegistryMemory = ptr("128Mi:512Mi")
	conf.RegistryCPU = ptr("200m")
	conf.BuildkitdMemory = ptr("256Mi:10Gi")
	conf.BuildkitdCPU = ptr("800m")
	conf.BuildkitdServiceMemory = ptr("128Mi:256Mi")
	conf.BuildkitdServiceCPU = ptr("200m")
	conf.ControllerMemory = ptr("256Mi")
	conf.ControllerCPU = ptr("100m")
	conf.APIServerMemory = ptr("256Mi")
	conf.APIServerCPU = ptr("100m")

	return conf
}
