package config

import (
	"encoding/json"

	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/meta"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
)

type Getter interface {
	Get(obj meta.Object, name string, opts *meta.GetOptions) error
}

type Config struct {
	IngressClassName            string   `json:"ingressClassName,omitempty"`
	ClusterDomains              []string `json:"clusterDomains,omitempty"`
	TLSEnabled                  bool     `json:"tlsEnabled,omitempty"`
	SetPodSecurityEnforeProfile *bool    `json:"setPodSecurityEnforeProfile,omitempty"`
	PodSecurityEnforceProfile   string   `json:"podSecurityEnforceProfile,omitempty"`
}

func (c *Config) complete() {
	if c.SetPodSecurityEnforeProfile == nil {
		c.SetPodSecurityEnforeProfile = &[]bool{true}[0]
	}
	if c.PodSecurityEnforceProfile == "" && *c.SetPodSecurityEnforeProfile {
		c.PodSecurityEnforceProfile = "baseline"
	}
}

func defaultConfig() *Config {
	cfg := &Config{}
	cfg.complete()
	return cfg
}

func Get(getter Getter) (*Config, error) {
	cm := &corev1.ConfigMap{}
	err := getter.Get(cm, system.ConfigName, &meta.GetOptions{
		Namespace: system.Namespace,
	})
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
