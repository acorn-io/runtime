package labels

import (
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/typed"
	"golang.org/x/exp/maps"
)

const (
	Prefix                       = "acorn.io/"
	AcornAppGeneration           = Prefix + "app-generation"
	AcornAppNamespace            = Prefix + "app-namespace"
	AcornAppName                 = Prefix + "app-name"
	AcornAcornName               = Prefix + "acorn-name"
	AcornServiceName             = Prefix + "service-name"
	AcornServicePublish          = Prefix + "service-publish"
	AcornServiceNamePrefix       = "service-name." + Prefix
	AcornDepNames                = Prefix + "dep-names"
	AcornAppUID                  = Prefix + "app-uid"
	AcornVolumeName              = Prefix + "volume-name"
	AcornSecretName              = Prefix + "secret-name"
	AcornSecretGenerated         = Prefix + "secret-generated"
	AcornContainerName           = Prefix + "container-name"
	AcornRouterName              = Prefix + "router-name"
	AcornJobName                 = Prefix + "job-name"
	AcornAppImage                = Prefix + "app-image"
	AcornAppCuePath              = Prefix + "app-cue-path"
	AcornAppCuePathHash          = Prefix + "app-cue-path-hash"
	AcornManaged                 = Prefix + "managed"
	AcornContainerSpec           = Prefix + "container-spec"
	AcornImageMapping            = Prefix + "image-mapping"
	AcornPortNumberPrefix        = "port-number." + Prefix
	AcornCredential              = Prefix + "credential"
	AcornPullSecret              = Prefix + "pull-secret"
	AcornSecretRevPrefix         = "secret-rev." + Prefix
	AcornPublishURL              = Prefix + "publish-url"
	AcornTargets                 = Prefix + "targets"
	AcornDNSHash                 = Prefix + "dns-hash"
	AcornLinkName                = Prefix + "link-name"
	AcornDNSState                = Prefix + "applied-dns-state"
	AcornDebugShell              = Prefix + "debug-shell"
	AcornDomain                  = Prefix + "domain"
	AcornCertNotValidBefore      = Prefix + "cert-not-valid-before"
	AcornCertNotValidAfter       = Prefix + "cert-not-valid-after"
	AcornLetsEncryptSettingsHash = Prefix + "le-hash"
)

func Merge(base, overlay map[string]string) map[string]string {
	result := maps.Clone(base)
	if result == nil {
		result = map[string]string{}
	}
	maps.Copy(result, overlay)
	return result
}

func ExcludeAcornKey(input map[string]string) map[string]string {
	result := map[string]string{}
	for k, v := range input {
		if strings.Contains(k, "acorn.io/") {
			continue
		}
		result[k] = v
	}
	return result
}

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

// GatherScoped takes in labels (or annotations) from the various places they can appear in an acorn app and sifts through
// them to build a map of the ones that apply to the resource specified by the supplied resourceName and resourceType.
// `globalLabels` would be labels defined at top of an Acornfile, as sibling to containers. These apply to all resources, thus "global."
// `resourceLabels` come from the specific resource in the Acornfile. We know all of these apply to the resource, by definition.
// `scoped` generally live on appInstance.Status.AppSpec and ultimately come from the user launching the acorn in the form of command line flags.
func GatherScoped(resourceName, resourceType string, globalLabels, resourceLabels map[string]string, scoped []v1.ScopedLabel) map[string]string {
	m := typed.Concat(globalLabels, resourceLabels)

	for _, scopedLabel := range scoped {
		if scopedLabel.ResourceType == "" {
			if scopedLabel.ResourceName == "" {
				m[scopedLabel.Key] = scopedLabel.Value
			} else if scopedLabel.ResourceName == resourceName {
				m[scopedLabel.Key] = scopedLabel.Value
			}
		} else if strings.EqualFold(scopedLabel.ResourceType, resourceType) {
			if scopedLabel.ResourceName == "" || scopedLabel.ResourceName == resourceName {
				m[scopedLabel.Key] = scopedLabel.Value
			}
		}
	}
	return ExcludeAcornKey(m)
}
