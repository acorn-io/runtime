package config

import (
	"context"
	"encoding/json"

	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
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
}

func defaultConfig() *Config {
	cfg := &Config{}
	cfg.complete()
	return cfg
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
	if err := json.Unmarshal([]byte(cm.Data["config"]), cm); err != nil {
		return nil, err
	}

	config.complete()
	return config, nil
}
