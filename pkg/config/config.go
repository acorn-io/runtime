package config

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ClusterDomainDefault         = ".local.on-acorn.io"
	InternalClusterDomainDefault = "svc.cluster.local"

	// AcornDNSEndpointDefault will be overridden at build time for releases
	AcornDNSEndpointDefault = "https://staging-dns.acrn.io/v1"
	AcornDNSStateDefault    = "auto"

	// Let's Encrypt
	LetsEncryptOptionDefault = "staging"
)

func complete(c *apiv1.Config, ctx context.Context, getter kclient.Reader) error {
	if c.TLSEnabled == nil {
		c.TLSEnabled = new(bool)
	}
	if len(c.DefaultPublishMode) == 0 {
		c.DefaultPublishMode = v1.PublishModeDefined
	}
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
	if *c.LetsEncrypt == "production" {
		if c.LetsEncryptEmail == "" {
			return fmt.Errorf("letsencrypt email is required when using production")
		}
		if !*c.LetsEncryptTOSAgree {
			return fmt.Errorf("letsencrypt TOS must be agreed to when using production")
		}
	}

	return nil
}

func setClusterDomains(ctx context.Context, c *apiv1.Config, getter kclient.Reader) error {
	useLocal, err := useLocalWildcardDomain(ctx, getter)
	if err != nil {
		return err
	}

	// Acorn DNS should be used if it is explicitly "enabled" or if it is in "auto" mode and the user hasn't set a
	//cluster domain and the cluster doesn't qualify for using the localhost wildcard domain
	if strings.EqualFold(*c.AcornDNS, "enabled") || (strings.EqualFold(*c.AcornDNS, "auto") && len(c.ClusterDomains) == 0 && !useLocal) {
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

	// If a clusterDomain hasn't been set yet, use the localhost wildcard domain
	if len(c.ClusterDomains) == 0 {
		c.ClusterDomains = []string{ClusterDomainDefault}
	}
	return nil
}

// If the cluster is a known desktop cluster type such as minikube, Rancher Desktop, or Docker Desktop, then we don't
// want to create real DNS records. Rather, use our wildcard domain that resolves to 127.0.0.1
func useLocalWildcardDomain(ctx context.Context, getter kclient.Reader) (bool, error) {
	var nodes corev1.NodeList
	if err := getter.List(ctx, &nodes); err != nil {
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

	if newConfig.IngressClassName != nil {
		if *newConfig.IngressClassName == "" {
			mergedConfig.IngressClassName = nil
		} else {
			mergedConfig.IngressClassName = newConfig.IngressClassName
		}
	}
	if newConfig.TLSEnabled != nil {
		mergedConfig.TLSEnabled = newConfig.TLSEnabled
	}
	if newConfig.SetPodSecurityEnforceProfile == nil {
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
	if len(newConfig.DefaultPublishMode) > 0 {
		mergedConfig.DefaultPublishMode = newConfig.DefaultPublishMode
	}
	if newConfig.AcornDNS != nil {
		mergedConfig.AcornDNS = newConfig.AcornDNS
	}
	if newConfig.AcornDNSEndpoint != nil {
		mergedConfig.AcornDNSEndpoint = newConfig.AcornDNSEndpoint
	}
	if newConfig.LetsEncrypt != nil {
		mergedConfig.LetsEncrypt = newConfig.LetsEncrypt
	}
	if newConfig.LetsEncryptEmail != "" {
		mergedConfig.LetsEncryptEmail = newConfig.LetsEncryptEmail
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

	config := &apiv1.Config{}
	if err := json.Unmarshal([]byte(cm.Data["config"]), config); err != nil {
		return nil, err
	}

	return config, nil
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

func Set(ctx context.Context, client kclient.Client, cfg *apiv1.Config) error {
	err := client.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: system.Namespace,
		},
	})
	if err != nil && !apierror.IsAlreadyExists(err) {
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
	err = complete(cfg, ctx, getter)
	return cfg, err
}
