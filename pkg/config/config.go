package config

import (
	"context"
	"encoding/json"

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
	ClusterDomainDefault = ".local.on-acorn.io"
)

func complete(c *apiv1.Config) {
	if c.TLSEnabled == nil {
		c.TLSEnabled = new(bool)
	}
	if len(c.PublishProtocolsByDefault) == 0 {
		c.PublishProtocolsByDefault = []string{string(v1.ProtocolHTTP)}
	}
	if c.SetPodSecurityEnforceProfile == nil {
		c.SetPodSecurityEnforceProfile = &[]bool{true}[0]
	}
	if c.PodSecurityEnforceProfile == "" && *c.SetPodSecurityEnforceProfile {
		c.PodSecurityEnforceProfile = "baseline"
	}
	if len(c.ClusterDomains) == 0 {
		c.ClusterDomains = []string{ClusterDomainDefault}
	}
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
		mergedConfig.IngressClassName = newConfig.IngressClassName
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
	if len(newConfig.ClusterDomains) > 0 && newConfig.ClusterDomains[0] == "" {
		mergedConfig.ClusterDomains = nil
	} else if len(newConfig.ClusterDomains) > 0 {
		mergedConfig.ClusterDomains = newConfig.ClusterDomains
	}
	if len(newConfig.PublishProtocolsByDefault) > 0 && newConfig.PublishProtocolsByDefault[0] == "" {
		mergedConfig.PublishProtocolsByDefault = nil
	} else if len(newConfig.PublishProtocolsByDefault) > 0 {
		mergedConfig.PublishProtocolsByDefault = newConfig.PublishProtocolsByDefault
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
	complete(cfg)
	return cfg, nil
}
