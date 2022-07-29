package labels

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"golang.org/x/exp/maps"
)

const (
	Prefix                 = "acorn.io/"
	AcornAppGeneration     = Prefix + "app-generation"
	AcornAppNamespace      = Prefix + "app-namespace"
	AcornAppName           = Prefix + "app-name"
	AcornAcornName         = Prefix + "acorn-name"
	AcornServiceName       = Prefix + "service-name"
	AcornServicePublish    = Prefix + "service-publish"
	AcornServiceNamePrefix = "service-name." + Prefix
	AcornDepNames          = Prefix + "dep-names"
	AcornAppUID            = Prefix + "app-uid"
	AcornVolumeName        = Prefix + "volume-name"
	AcornSecretName        = Prefix + "secret-name"
	AcornSecretGenerated   = Prefix + "secret-generated"
	AcornContainerName     = Prefix + "container-name"
	AcornJobName           = Prefix + "job-name"
	AcornAppImage          = Prefix + "app-image"
	AcornAppCuePath        = Prefix + "app-cue-path"
	AcornAppCuePathHash    = Prefix + "app-cue-path-hash"
	AcornManaged           = Prefix + "managed"
	AcornContainerSpec     = Prefix + "container-spec"
	AcornImageMapping      = Prefix + "image-mapping"
	AcornPortNumberPrefix  = "port-number." + Prefix
	AcornCredential        = Prefix + "credential"
	AcornPullSecret        = Prefix + "pull-secret"
	AcornSecretRevPrefix   = "secret-rev." + Prefix
	AcornRootNamespace     = Prefix + "root-namespace"
	AcornRootPrefix        = Prefix + "root-prefix"
	AcornPublishURL        = Prefix + "publish-url"
	AcornTargets           = Prefix + "targets"
	AcornDNSHash           = Prefix + "dns-hash"
	AcornLinkName          = Prefix + "link-name"
	AcornDNSState          = Prefix + "applied-dns-state"
)

func RootPrefix(parentLabels map[string]string, name string) string {
	prefix := parentLabels[AcornRootPrefix]
	if prefix == "" {
		prefix = name
	} else {
		prefix += "." + name
	}
	return prefix
}

func Merge(base, overlay map[string]string) map[string]string {
	result := maps.Clone(base)
	if result == nil {
		result = map[string]string{}
	}
	maps.Copy(result, overlay)
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
