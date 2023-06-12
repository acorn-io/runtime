package labels

import (
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/typed"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

const (
	Prefix                              = "acorn.io/"
	AcornAccountID                      = Prefix + "account-id"
	AcornAppGeneration                  = Prefix + "app-generation"
	AcornAppNamespace                   = Prefix + "app-namespace"
	AcornAppName                        = Prefix + "app-name"
	AcornParentAcornName                = Prefix + "parent-acorn-name"
	AcornAppPublicName                  = Prefix + "app-public-name"
	AcornPublicName                     = Prefix + "public-name"
	AcornAcornName                      = Prefix + "acorn-name"
	AcornServiceName                    = Prefix + "service-name"
	AcornServicePublish                 = Prefix + "service-publish"
	AcornServiceNamePrefix              = "service-name." + Prefix
	AcornDepNames                       = Prefix + "dep-names"
	AcornAppUID                         = Prefix + "app-uid"
	AcornVolumeName                     = Prefix + "volume-name"
	AcornVolumeClass                    = Prefix + "volume-class"
	AcornSecretName                     = Prefix + "secret-name"
	AcornSecretSourceName               = Prefix + "secret-source-name"
	AcornSecretGenerated                = Prefix + "secret-generated"
	AcornContainerName                  = Prefix + "container-name"
	AcornRouterName                     = Prefix + "router-name"
	AcornJobName                        = Prefix + "job-name"
	AcornAppImage                       = Prefix + "app-image"
	AcornAppDevHash                     = Prefix + "app-dev-hash"
	AcornManaged                        = Prefix + "managed"
	AcornContainerSpec                  = Prefix + "container-spec"
	AcornImageMapping                   = Prefix + "image-mapping"
	AcornOriginalImage                  = Prefix + "original-image"
	AcornPortNumberPrefix               = "port-number." + Prefix
	AcornCredential                     = Prefix + "credential"
	AcornPullSecret                     = Prefix + "pull-secret"
	AcornSecretRevPrefix                = "secret-rev." + Prefix
	AcornPublishURL                     = Prefix + "publish-url"
	AcornTargets                        = Prefix + "targets"
	AcornDNSHash                        = Prefix + "dns-hash"
	AcornLinkName                       = Prefix + "link-name"
	AcornDNSState                       = Prefix + "applied-dns-state"
	AcornDomain                         = Prefix + "domain"
	AcornCertNotValidBefore             = Prefix + "cert-not-valid-before"
	AcornCertNotValidAfter              = Prefix + "cert-not-valid-after"
	AcornLetsEncryptSettingsHash        = Prefix + "le-hash"
	AcornProject                        = Prefix + "project"
	AcornProjectName                    = Prefix + "project-name"
	AcornProjectDefaultRegion           = Prefix + "project-default-region"
	AcornProjectSupportedRegions        = Prefix + "project-supported-regions"
	AcornCalculatedProjectDefaultRegion = Prefix + "calculated-project-default-region"
	ProjectEnforcedQuotaAnnotation      = Prefix + "enforced-quota"
	AcornPermissions                    = Prefix + "permissions"
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

func ManagedByApp(appNamespace, appName string, kv ...string) map[string]string {
	labels := map[string]string{
		AcornAppName:      appName,
		AcornAppNamespace: appNamespace,
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

func Managed(appInstance *v1.AppInstance, kv ...string) map[string]string {
	return ManagedByApp(appInstance.Namespace, appInstance.Name, kv...)
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

func FilterUserDefined(appInstance *v1.AppInstance, allowedLabels, allowedAnnotations []string) *v1.AppInstance {
	appInstance.Spec.Labels = filterScoped(appInstance.Spec.Labels, allowedLabels)
	appInstance.Spec.Annotations = filterScoped(appInstance.Spec.Annotations, allowedAnnotations)

	appInstance.Status.AppSpec.Labels = filter(appInstance.Status.AppSpec.Labels, allowedLabels)
	appInstance.Status.AppSpec.Annotations = filter(appInstance.Status.AppSpec.Annotations, allowedAnnotations)

	for key, c := range appInstance.Status.AppSpec.Containers {
		c.Labels = filter(c.Labels, allowedLabels)
		c.Annotations = filter(c.Annotations, allowedAnnotations)
		appInstance.Status.AppSpec.Containers[key] = c
	}

	for key, j := range appInstance.Status.AppSpec.Jobs {
		j.Labels = filter(j.Labels, allowedLabels)
		j.Annotations = filter(j.Annotations, allowedAnnotations)
		appInstance.Status.AppSpec.Jobs[key] = j
	}

	for key, r := range appInstance.Status.AppSpec.Routers {
		r.Labels = filter(r.Labels, allowedLabels)
		r.Annotations = filter(r.Annotations, allowedAnnotations)
		appInstance.Status.AppSpec.Routers[key] = r
	}

	for key, v := range appInstance.Status.AppSpec.Volumes {
		v.Labels = filter(v.Labels, allowedLabels)
		v.Annotations = filter(v.Annotations, allowedAnnotations)
		appInstance.Status.AppSpec.Volumes[key] = v
	}

	for key, s := range appInstance.Status.AppSpec.Services {
		labels := v1.ScopedLabels{}
		annotations := v1.ScopedLabels{}
		for _, v := range s.Labels {
			if slices.Contains(allowedLabels, v.Key) {
				labels = append(labels, v)
			}
		}
		for _, v := range s.Annotations {
			if slices.Contains(allowedAnnotations, v.Key) {
				annotations = append(annotations, v)
			}
		}
		s.Labels = labels
		s.Annotations = annotations
		appInstance.Status.AppSpec.Services[key] = s
	}

	return appInstance
}

func filterScoped(scoped []v1.ScopedLabel, allowed []string) []v1.ScopedLabel {
	if len(allowed) == 0 {
		// If nothing is allowed, then short-circuit
		return nil
	}

	result := make([]v1.ScopedLabel, 0, len(allowed))
	for _, label := range scoped {
		if slices.Contains(allowed, label.Key) {
			result = append(result, label)
		}
	}

	return result
}

func filter[K comparable, V any](m map[K]V, allowed []K) map[K]V {
	if len(allowed) == 0 {
		// If nothing is allowed, then short-circuit
		return nil
	}

	result := make(map[K]V, len(allowed))
	for _, key := range allowed {
		if val, ok := m[key]; ok {
			result[key] = val
		}
	}
	return result
}
