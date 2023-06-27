package config

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/acorn-io/baaah/pkg/router"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/system"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ClusterDomainDefault         = ".local.oss-acorn.io"
	InternalClusterDomainDefault = "svc.cluster.local"

	AcornDNSEndpointDefault = "https://oss-dns.acrn.io/v1"
	AcornDNSStateDefault    = "auto"

	// LetsEncryptOptionDefault is the default state for the Let's Encrypt integration
	LetsEncryptOptionDefault = "disabled"

	// DefaultImageCheckIntervalDefault is the default value for the DefaultImageCheckInterval field
	DefaultImageCheckIntervalDefault = "5m"

	// Default HttpEndpointPattern set to enable Let's Encrypt
	DefaultHttpEndpointPattern = "{{hashConcat 8 .Container .App .Namespace | truncate}}.{{.ClusterDomain}}"

	// Features
	FeatureImageAllowRules = "image-allow-rules"
	FeatureDefaults        = map[string]bool{
		FeatureImageAllowRules: false,
	}
)

func complete(ctx context.Context, c *apiv1.Config, getter kclient.Reader) error {
	if c.SetPodSecurityEnforceProfile == nil {
		c.SetPodSecurityEnforceProfile = &[]bool{true}[0]
	}
	if c.PodSecurityEnforceProfile == "" && *c.SetPodSecurityEnforceProfile {
		c.PodSecurityEnforceProfile = "baseline"
	}
	if c.AcornDNS == nil {
		c.AcornDNS = &AcornDNSStateDefault
	}
	if c.AcornDNSEndpoint == nil || *c.AcornDNSEndpoint == "" {
		c.AcornDNSEndpoint = &AcornDNSEndpointDefault
	}
	err := setClusterDomains(ctx, c, getter)
	if err != nil {
		return err
	}
	if c.InternalClusterDomain == "" {
		c.InternalClusterDomain = InternalClusterDomainDefault
	}
	if c.LetsEncrypt == nil {
		c.LetsEncrypt = &LetsEncryptOptionDefault
	}
	if c.LetsEncryptTOSAgree == nil {
		c.LetsEncryptTOSAgree = new(bool)
	}
	if c.AutoUpgradeInterval == nil || *c.AutoUpgradeInterval == "" {
		c.AutoUpgradeInterval = &DefaultImageCheckIntervalDefault
	}
	if c.RecordBuilds == nil {
		c.RecordBuilds = new(bool)
	}
	if c.PublishBuilders == nil {
		c.PublishBuilders = new(bool)
	}
	if c.BuilderPerProject == nil {
		c.BuilderPerProject = new(bool)
	}
	if c.HttpEndpointPattern == nil || *c.HttpEndpointPattern == "" {
		c.HttpEndpointPattern = &DefaultHttpEndpointPattern
	}
	if c.WorkloadMemoryDefault == nil {
		c.WorkloadMemoryDefault = new(int64)
	}
	if c.WorkloadMemoryMaximum == nil {
		c.WorkloadMemoryMaximum = new(int64)
	}
	if c.InternalRegistryPrefix == nil {
		c.InternalRegistryPrefix = new(string)
	}
	if c.IgnoreUserLabelsAndAnnotations == nil {
		c.IgnoreUserLabelsAndAnnotations = new(bool)
	}
	if c.ManageVolumeClasses == nil {
		c.ManageVolumeClasses = new(bool)
	}
	if c.UseCustomCABundle == nil {
		c.UseCustomCABundle = new(bool)
	}
	if c.NetworkPolicies == nil {
		c.NetworkPolicies = new(bool)
	}
	if c.IngressControllerNamespace == nil {
		c.IngressControllerNamespace = new(string)
	}
	if c.AWSIdentityProviderARN == nil {
		c.AWSIdentityProviderARN = new(string)
	}
	if c.Features == nil {
		c.Features = FeatureDefaults
	} else {
		for k, v := range FeatureDefaults {
			if _, ok := c.Features[k]; !ok {
				c.Features[k] = v
			}
		}
	}
	if c.CertManagerIssuer == nil {
		c.CertManagerIssuer = new(string)
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
		c.ClusterDomains = []string{ClusterDomainDefault}
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
// are two main cases to be considered when adding a new field to the Config object and merging it here:
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

	return config, complete(ctx, config, getter)
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
	return newConfig, complete(ctx, newConfig, client)
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
	err = complete(ctx, cfg, getter)
	return cfg, err
}
