package config

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/acorn-io/baaah/pkg/router"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/profiles"
	"github.com/acorn-io/runtime/pkg/system"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func devConfig(ctx context.Context, c *apiv1.Config, getter kclient.Reader) error {
	var devConfig corev1.ConfigMap
	if err := getter.Get(ctx, router.Key(system.Namespace, system.DevConfigName), &devConfig); apierror.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	devCfg, err := unmarshal(&devConfig)
	if err != nil {
		return err
	}

	*c = *merge(c, devCfg)
	return nil
}

func complete(ctx context.Context, c *apiv1.Config, getter kclient.Reader, includeDev bool) error {
	if includeDev {
		if err := devConfig(ctx, c, getter); err != nil {
			return err
		}
	}

	profile := profiles.Get(c.Profile)
	if c.SetPodSecurityEnforceProfile == nil {
		c.SetPodSecurityEnforceProfile = profile.SetPodSecurityEnforceProfile
	}
	if c.PodSecurityEnforceProfile == "" && *c.SetPodSecurityEnforceProfile {
		c.PodSecurityEnforceProfile = profile.PodSecurityEnforceProfile
	}
	if c.AcornDNS == nil {
		c.AcornDNS = profile.AcornDNS
	}
	if c.AcornDNSEndpoint == nil || *c.AcornDNSEndpoint == "" {
		c.AcornDNSEndpoint = profile.AcornDNSEndpoint
	}
	err := setClusterDomains(ctx, c, getter)
	if err != nil {
		return err
	}
	if c.InternalClusterDomain == "" {
		c.InternalClusterDomain = profile.InternalClusterDomain
	}
	if c.LetsEncrypt == nil {
		c.LetsEncrypt = profile.LetsEncrypt
	}
	if c.LetsEncryptTOSAgree == nil {
		c.LetsEncryptTOSAgree = profile.LetsEncryptTOSAgree
	}
	if c.AutoUpgradeInterval == nil || *c.AutoUpgradeInterval == "" {
		c.AutoUpgradeInterval = profile.AutoUpgradeInterval
	}
	if c.RecordBuilds == nil {
		c.RecordBuilds = profile.RecordBuilds
	}
	if c.PublishBuilders == nil {
		c.PublishBuilders = profile.PublishBuilders
	}
	if c.BuilderPerProject == nil {
		c.BuilderPerProject = profile.BuilderPerProject
	}
	if c.HttpEndpointPattern == nil || *c.HttpEndpointPattern == "" {
		c.HttpEndpointPattern = profile.HttpEndpointPattern
	}
	if c.WorkloadMemoryDefault == nil {
		c.WorkloadMemoryDefault = profile.WorkloadMemoryDefault
	}
	if c.WorkloadMemoryMaximum == nil {
		c.WorkloadMemoryMaximum = profile.WorkloadMemoryMaximum
	}
	if c.InternalRegistryPrefix == nil {
		c.InternalRegistryPrefix = profile.InternalRegistryPrefix
	}
	if c.IgnoreUserLabelsAndAnnotations == nil {
		c.IgnoreUserLabelsAndAnnotations = profile.IgnoreUserLabelsAndAnnotations
	}
	if c.ManageVolumeClasses == nil {
		c.ManageVolumeClasses = profile.ManageVolumeClasses
	}
	if c.UseCustomCABundle == nil {
		c.UseCustomCABundle = profile.UseCustomCABundle
	}
	if c.NetworkPolicies == nil {
		c.NetworkPolicies = profile.NetworkPolicies
	}
	if c.IngressControllerNamespace == nil {
		c.IngressControllerNamespace = profile.IngressControllerNamespace
	}
	if c.AWSIdentityProviderARN == nil {
		c.AWSIdentityProviderARN = profile.AWSIdentityProviderARN
	}
	if c.RegistryMemory == nil {
		c.RegistryMemory = profile.RegistryMemory
	}
	if c.RegistryCPU == nil {
		c.RegistryCPU = profile.RegistryCPU
	}
	if c.BuildkitdMemory == nil {
		c.BuildkitdMemory = profile.BuildkitdMemory
	}
	if c.BuildkitdCPU == nil {
		c.BuildkitdCPU = profile.BuildkitdCPU
	}
	if c.BuildkitdServiceMemory == nil {
		c.BuildkitdServiceMemory = profile.BuildkitdServiceMemory
	}
	if c.BuildkitdServiceCPU == nil {
		c.BuildkitdServiceCPU = profile.BuildkitdServiceCPU
	}
	if c.ControllerMemory == nil {
		c.ControllerMemory = profile.ControllerMemory
	}
	if c.ControllerCPU == nil {
		c.ControllerCPU = profile.ControllerCPU
	}
	if c.APIServerMemory == nil {
		c.APIServerMemory = profile.APIServerMemory
	}
	if c.APIServerCPU == nil {
		c.APIServerCPU = profile.APIServerCPU
	}
	if c.Features == nil {
		c.Features = profile.Features
	} else {
		for k, v := range profiles.FeatureDefaults {
			if _, ok := c.Features[k]; !ok {
				c.Features[k] = v
			}
		}
	}
	if c.CertManagerIssuer == nil {
		c.CertManagerIssuer = profile.CertManagerIssuer
	}
	return nil
}

// shouldLookupAcornDNSDomain determines if given the current configuration, Acorn DNS domain should be used if
// found. Extra care is taken to ensure we only do extra API object lookups when necessary. Most importantly some objects
// like v1.Node won't exist in manager and will fail there, so there should be a user configuration that will make lookups
// not happen.
func shouldLookupAcornDNSDomain(ctx context.Context, c *apiv1.Config, getter kclient.Reader) (bool, error) {
	if strings.EqualFold(*c.AcornDNS, "enabled") {
		// if acorn dns is enabled then we know we have to lookup
		return true, nil
	}
	if !strings.EqualFold(*c.AcornDNS, "auto") {
		// if acorn dns is not auto, then it must be disabled, so we know we don't need to lookup
		return false, nil
	}
	if len(c.ClusterDomains) > 0 {
		// The only acorn dns option left is "auto" and if the user has set a cluster domain then
		// by definition of what "auto" is we shouldn't lookup the acorn dns domain
		return false, nil
	}
	// at this point the user has selected acorn dns "auto" and there are no cluster domains set, so now we
	// do any additional lookup to determine if the localhost DNS should be used
	useLocal, err := useLocalWildcardDomain(ctx, getter)

	// only lookup acorn dns domain if we don't want to use localhost DNS
	return !useLocal, err
}

func setClusterDomains(ctx context.Context, c *apiv1.Config, getter kclient.Reader) error {
	shouldLookupAcornDNSDomain, err := shouldLookupAcornDNSDomain(ctx, c, getter)
	if err != nil {
		return err
	}

	// Acorn DNS should be used if it is explicitly "enabled" or if it is in "auto" mode and the user hasn't set a
	// cluster domain and the cluster doesn't qualify for using the localhost wildcard domain
	if shouldLookupAcornDNSDomain {
		dnsSecret := &corev1.Secret{}
		err = getter.Get(ctx, router.Key(system.Namespace, system.DNSSecretName), dnsSecret)
		if err != nil && !apierror.IsNotFound(err) {
			return err
		}
		domain := string(dnsSecret.Data["domain"])
		if domain != "" {
			c.ClusterDomains = append(c.ClusterDomains, domain)
		}
	}

	// If a clusterDomain hasn't been set yet and acorn-dns hasn't been explicitly disabled,
	// use the localhost wildcard domain
	if len(c.ClusterDomains) == 0 && !strings.EqualFold(*c.AcornDNS, "disabled") {
		c.ClusterDomains = []string{profiles.ClusterDomainDefault}
	}
	return nil
}

// If the cluster is a known desktop cluster type such as minikube, Rancher Desktop, or Docker Desktop, then we don't
// want to create real DNS records. Rather, use our wildcard domain that resolves to 127.0.0.1
func useLocalWildcardDomain(ctx context.Context, getter kclient.Reader) (bool, error) {
	var nodes corev1.NodeList
	if err := getter.List(ctx, &nodes); err != nil {
		if meta.IsNoMatchError(err) {
			// Node type doesn't exist probably because we are running against manager
			return false, nil
		}
		return false, err
	}

	if len(nodes.Items) == 1 {
		node := nodes.Items[0]
		if strings.Contains(node.Name, "rancher-desktop") || strings.Contains(node.Status.NodeInfo.OSImage, "Rancher Desktop") ||
			node.Name == "docker-desktop" || strings.Contains(node.Name, "minikube") {
			return true, nil
		}
	}

	// Look for k3d
	for _, node := range nodes.Items {
		if strings.HasPrefix(node.Spec.ProviderID, "k3s://k3d") {
			return true, nil
		}
	}

	return false, nil
}

func IsDockerDesktop(ctx context.Context, getter kclient.Reader) (bool, error) {
	var nodes corev1.NodeList
	if err := getter.List(ctx, &nodes); err != nil {
		return false, err
	}

	if len(nodes.Items) == 1 {
		node := nodes.Items[0]
		if node.Name == "docker-desktop" {
			return true, nil
		}
	}

	return false, nil
}

// merge merges two Config objects. The newConfig object takes precedence over the oldConfig object.
//
// WARNING: We have had many bugs with this merge logic. To avoid this when adding fields here, there
// are three main cases to be considered when adding a new field to the Config object and merging it here:
//
// 1. If the newConfig does not pass a field at all, the field in the oldConfig should be used.
// 2. The newConfig should have a way of unsetting the values in the oldConfig.
// 3. A newConfig should have a way of setting values that overwrite the oldConfig.
func merge(oldConfig, newConfig *apiv1.Config) *apiv1.Config {
	var (
		mergedConfig apiv1.Config
	)

	if oldConfig != nil {
		mergedConfig = *oldConfig
	}
	if newConfig == nil {
		return &mergedConfig
	}

	if newConfig.IgnoreUserLabelsAndAnnotations != nil {
		mergedConfig.IgnoreUserLabelsAndAnnotations = newConfig.IgnoreUserLabelsAndAnnotations
	}

	if newConfig.ManageVolumeClasses != nil {
		mergedConfig.ManageVolumeClasses = newConfig.ManageVolumeClasses
	}

	if newConfig.Profile != nil {
		mergedConfig.Profile = newConfig.Profile
	}

	// This is to provide a way to reset value to empty if user passes --flag "" as empty string
	if len(newConfig.AllowUserAnnotations) > 0 && newConfig.AllowUserAnnotations[0] == "" {
		mergedConfig.AllowUserAnnotations = nil
	} else if len(newConfig.AllowUserAnnotations) > 0 {
		mergedConfig.AllowUserAnnotations = newConfig.AllowUserAnnotations
	}

	if len(newConfig.AllowUserLabels) > 0 && newConfig.AllowUserLabels[0] == "" {
		mergedConfig.AllowUserLabels = nil
	} else if len(newConfig.AllowUserLabels) > 0 {
		mergedConfig.AllowUserLabels = newConfig.AllowUserLabels
	}

	if newConfig.IngressClassName != nil {
		if *newConfig.IngressClassName == "" {
			mergedConfig.IngressClassName = nil
		} else {
			mergedConfig.IngressClassName = newConfig.IngressClassName
		}
	}
	if newConfig.SetPodSecurityEnforceProfile != nil {
		mergedConfig.SetPodSecurityEnforceProfile = newConfig.SetPodSecurityEnforceProfile
	}
	if newConfig.PodSecurityEnforceProfile != "" {
		mergedConfig.PodSecurityEnforceProfile = newConfig.PodSecurityEnforceProfile
	}
	if newConfig.InternalClusterDomain != "" {
		mergedConfig.InternalClusterDomain = newConfig.InternalClusterDomain
	}
	if len(newConfig.ClusterDomains) > 0 && newConfig.ClusterDomains[0] == "" {
		mergedConfig.ClusterDomains = nil
	} else if len(newConfig.ClusterDomains) > 0 {
		for i, cd := range newConfig.ClusterDomains {
			if !strings.HasPrefix(cd, ".") {
				newConfig.ClusterDomains[i] = "." + cd
			}
		}
		mergedConfig.ClusterDomains = newConfig.ClusterDomains
	}
	if newConfig.HttpEndpointPattern != nil {
		mergedConfig.HttpEndpointPattern = newConfig.HttpEndpointPattern
	}
	if newConfig.AcornDNS != nil {
		mergedConfig.AcornDNS = newConfig.AcornDNS
	}
	if newConfig.AcornDNSEndpoint != nil {
		mergedConfig.AcornDNSEndpoint = newConfig.AcornDNSEndpoint
	}
	if newConfig.LetsEncryptTOSAgree != nil {
		mergedConfig.LetsEncryptTOSAgree = newConfig.LetsEncryptTOSAgree
	}
	if newConfig.LetsEncrypt != nil {
		mergedConfig.LetsEncrypt = newConfig.LetsEncrypt
	}
	if newConfig.LetsEncryptEmail != "" {
		mergedConfig.LetsEncryptEmail = newConfig.LetsEncryptEmail
	}
	if newConfig.AutoUpgradeInterval != nil {
		mergedConfig.AutoUpgradeInterval = newConfig.AutoUpgradeInterval
	}
	if newConfig.RecordBuilds != nil {
		mergedConfig.RecordBuilds = newConfig.RecordBuilds
	}
	if newConfig.PublishBuilders != nil {
		mergedConfig.PublishBuilders = newConfig.PublishBuilders
	}
	if newConfig.BuilderPerProject != nil {
		mergedConfig.BuilderPerProject = newConfig.BuilderPerProject
	}
	if newConfig.InternalRegistryPrefix != nil {
		mergedConfig.InternalRegistryPrefix = newConfig.InternalRegistryPrefix
	}
	if newConfig.WorkloadMemoryDefault != nil {
		mergedConfig.WorkloadMemoryDefault = newConfig.WorkloadMemoryDefault
	}
	if newConfig.WorkloadMemoryMaximum != nil {
		mergedConfig.WorkloadMemoryMaximum = newConfig.WorkloadMemoryMaximum
	}
	if newConfig.UseCustomCABundle != nil {
		mergedConfig.UseCustomCABundle = newConfig.UseCustomCABundle
	}
	if newConfig.NetworkPolicies != nil {
		mergedConfig.NetworkPolicies = newConfig.NetworkPolicies
	}
	if newConfig.IngressControllerNamespace != nil {
		mergedConfig.IngressControllerNamespace = newConfig.IngressControllerNamespace
	}
	if newConfig.AWSIdentityProviderARN != nil {
		mergedConfig.AWSIdentityProviderARN = newConfig.AWSIdentityProviderARN
	}
	if newConfig.EventTTL != nil {
		mergedConfig.EventTTL = newConfig.EventTTL
	}
	if newConfig.CertManagerIssuer != nil {
		mergedConfig.CertManagerIssuer = newConfig.CertManagerIssuer
	}
	if newConfig.Features != nil {
		mergedConfig.Features = newConfig.Features
	}

	if len(newConfig.PropagateProjectAnnotations) > 0 && newConfig.PropagateProjectAnnotations[0] == "" {
		mergedConfig.PropagateProjectAnnotations = nil
	} else if len(newConfig.PropagateProjectAnnotations) > 0 {
		mergedConfig.PropagateProjectAnnotations = newConfig.PropagateProjectAnnotations
	}

	if len(newConfig.PropagateProjectLabels) > 0 && newConfig.PropagateProjectLabels[0] == "" {
		mergedConfig.PropagateProjectLabels = nil
	} else if len(newConfig.PropagateProjectLabels) > 0 {
		mergedConfig.PropagateProjectLabels = newConfig.PropagateProjectLabels
	}

	if len(newConfig.AllowTrafficFromNamespace) > 0 && newConfig.AllowTrafficFromNamespace[0] == "" {
		mergedConfig.AllowTrafficFromNamespace = nil
	} else if len(newConfig.AllowTrafficFromNamespace) > 0 {
		mergedConfig.AllowTrafficFromNamespace = newConfig.AllowTrafficFromNamespace
	}

	if len(newConfig.ServiceLBAnnotations) > 0 && newConfig.ServiceLBAnnotations[0] == "" {
		mergedConfig.ServiceLBAnnotations = nil
	} else if len(newConfig.ServiceLBAnnotations) > 0 {
		mergedConfig.ServiceLBAnnotations = newConfig.ServiceLBAnnotations
	}

	if newConfig.RegistryMemory != nil {
		mergedConfig.RegistryMemory = newConfig.RegistryMemory
	}
	if newConfig.RegistryCPU != nil {
		mergedConfig.RegistryCPU = newConfig.RegistryCPU
	}
	if newConfig.BuildkitdMemory != nil {
		mergedConfig.BuildkitdMemory = newConfig.BuildkitdMemory
	}
	if newConfig.BuildkitdCPU != nil {
		mergedConfig.BuildkitdCPU = newConfig.BuildkitdCPU
	}
	if newConfig.BuildkitdServiceMemory != nil {
		mergedConfig.BuildkitdServiceMemory = newConfig.BuildkitdServiceMemory
	}
	if newConfig.BuildkitdServiceCPU != nil {
		mergedConfig.BuildkitdServiceCPU = newConfig.BuildkitdServiceCPU
	}
	if newConfig.ControllerMemory != nil {
		mergedConfig.ControllerMemory = newConfig.ControllerMemory
	}
	if newConfig.ControllerCPU != nil {
		mergedConfig.ControllerCPU = newConfig.ControllerCPU
	}
	if newConfig.APIServerMemory != nil {
		mergedConfig.APIServerMemory = newConfig.APIServerMemory
	}
	if newConfig.APIServerCPU != nil {
		mergedConfig.APIServerCPU = newConfig.APIServerCPU
	}

	return &mergedConfig
}

func Init(ctx context.Context, client kclient.Client) error {
	cm := &corev1.ConfigMap{}
	err := client.Get(ctx, router.Key(system.Namespace, system.ConfigName), cm)
	if apierror.IsNotFound(err) {
		return client.Create(ctx, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      system.ConfigName,
				Namespace: system.Namespace,
			},
			Data: map[string]string{"config": "{}"},
		})
	}
	return err
}

func Incomplete(ctx context.Context, getter kclient.Reader) (*apiv1.Config, error) {
	cm := &corev1.ConfigMap{}
	err := getter.Get(ctx, router.Key(system.Namespace, system.ConfigName), cm)
	if apierror.IsNotFound(err) {
		return &apiv1.Config{}, nil
	} else if err != nil {
		return nil, err
	}

	return unmarshal(cm)
}

func UnmarshalAndComplete(ctx context.Context, cm *corev1.ConfigMap, getter kclient.Reader) (*apiv1.Config, error) {
	config, err := unmarshal(cm)
	if err != nil {
		return nil, err
	}

	return config, complete(ctx, config, getter, true)
}

func unmarshal(cm *corev1.ConfigMap) (*apiv1.Config, error) {
	config := new(apiv1.Config)
	if len(cm.Data["config"]) == 0 {
		return config, nil
	}

	return config, json.Unmarshal([]byte(cm.Data["config"]), config)
}

func AsConfigMap(cfg *apiv1.Config) (*corev1.ConfigMap, error) {
	return asConfigMap(nil, cfg)
}

func asConfigMap(existing, cfg *apiv1.Config) (*corev1.ConfigMap, error) {
	newConfig := merge(existing, cfg)

	configBytes, err := json.Marshal(newConfig)
	if err != nil {
		return nil, err
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      system.ConfigName,
			Namespace: system.Namespace,
		},
		Data: map[string]string{
			"config": string(configBytes),
		},
		BinaryData: nil,
	}, nil
}

// TestSetGet will do everything that Set does, but instead of persisting the resulting config it will
// return the merged and completed config.  This is as though you did Set() followed by Get() except that the
// state in Kubernetes will not actually change.
func TestSetGet(ctx context.Context, client kclient.Client, cfg *apiv1.Config) (*apiv1.Config, error) {
	existing, err := Incomplete(ctx, client)
	if err != nil {
		return nil, err
	}

	newConfig := merge(existing, cfg)
	return newConfig, complete(ctx, newConfig, client, false)
}

func Set(ctx context.Context, client kclient.Client, cfg *apiv1.Config) error {
	err := client.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: system.Namespace,
		},
	})
	if err != nil && !apierror.IsAlreadyExists(err) && !meta.IsNoMatchError(err) {
		return err
	}

	existing, err := Incomplete(ctx, client)
	if err != nil {
		return err
	}

	configMap, err := asConfigMap(existing, cfg)
	if err != nil {
		return err
	}

	err = client.Update(ctx, configMap)
	if apierror.IsNotFound(err) {
		return client.Create(ctx, configMap)
	}
	return err
}

func Get(ctx context.Context, getter kclient.Reader) (*apiv1.Config, error) {
	cfg, err := Incomplete(ctx, getter)
	if err != nil {
		return nil, err
	}
	err = complete(ctx, cfg, getter, true)
	return cfg, err
}
