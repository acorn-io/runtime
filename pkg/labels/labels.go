package labels

import v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"

const (
	Prefix               = "acorn.io/"
	AcornAppNamespace    = Prefix + "app-namespace"
	AcornAppName         = Prefix + "app-name"
	AcornAcornName       = Prefix + "acorn-name"
	AcornAppUID          = Prefix + "app-uid"
	AcornVolumeName      = Prefix + "volume-name"
	AcornSecretName      = Prefix + "secret-name"
	AcornSecretGenerated = Prefix + "secret-generated"
	AcornContainerName   = Prefix + "container-name"
	AcornJobName         = Prefix + "job-name"
	AcornAppImage        = Prefix + "app-image"
	AcornAppCuePath      = Prefix + "app-cue-path"
	AcornAppCuePathHash  = Prefix + "app-cue-path-hash"
	AcornManaged         = Prefix + "managed"
	AcornContainerSpec   = Prefix + "container-spec"
	AcornImageMapping    = Prefix + "image-mapping"
	AcornPortNumber      = Prefix + "port-number"
	AcornHostnames       = Prefix + "hostnames"
	AcornAlias           = "alias." + Prefix
	AcornChildNamespaces = Prefix + "child-namespaces"
	AcornCredential      = Prefix + "credential"
	AcornPullSecret      = Prefix + "pull-secret"
	AcornSecretRevPrefix = "secret-rev." + Prefix
)

func Managed(appInstance *v1.AppInstance, kv ...string) map[string]string {
	labels := map[string]string{
		AcornAppName:      appInstance.Name,
		AcornAppNamespace: appInstance.Namespace,
		AcornManaged:      "true",
	}
	for i := 0; i+1 < len(kv); i += 2 {
		if kv[i+1] == "" {
			delete(labels, kv[i])
		} else {
			labels[kv[i]] = kv[i+1]
		}
	}
	return labels
}
