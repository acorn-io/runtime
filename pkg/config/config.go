package config

import (
	"context"
	"encoding/json"

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

type Config struct {
	IngressClassName             string   `json:"ingressClassName,omitempty"`
	ClusterDomains               []string `json:"clusterDomains,omitempty"`
	TLSEnabled                   bool     `json:"tlsEnabled,omitempty"`
	SetPodSecurityEnforceProfile *bool    `json:"setPodSecurityEnforceProfile,omitempty"`
	PodSecurityEnforceProfile    string   `json:"podSecurityEnforceProfile,omitempty"`
}

func (c *Config) complete() {
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

func defaultConfig() *Config {
	cfg := &Config{}
	cfg.complete()
	return cfg
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

func Get(ctx context.Context, getter kclient.Reader) (*Config, error) {
	cm := &corev1.ConfigMap{}
	err := getter.Get(ctx, router.Key(system.Namespace, system.ConfigName), cm)
	if apierror.IsNotFound(err) {
		return defaultConfig(), nil
	} else if err != nil {
		return nil, err
	}

	config := &Config{}
	if err := json.Unmarshal([]byte(cm.Data["config"]), config); err != nil {
		return nil, err
	}

	config.complete()
	return config, nil
}
