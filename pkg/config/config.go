package config

import (
	"encoding/json"

	"github.com/ibuildthecloud/baaah/pkg/meta"
	"github.com/ibuildthecloud/herd/pkg/system"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
)

type Getter interface {
	Get(obj meta.Object, name string, opts *meta.GetOptions) error
}

type Config struct {
	IngressClassName string   `json:"ingressClassName,omitempty"`
	ClusterDomains   []string `json:"clusterDomains,omitempty"`
}

func Get(getter Getter) (*Config, error) {
	cm := &corev1.ConfigMap{}
	err := getter.Get(cm, system.ConfigName, &meta.GetOptions{
		Namespace: system.Namespace,
	})
	if apierror.IsNotFound(err) {
		return &Config{}, nil
	} else if err != nil {
		return nil, err
	}

	config := &Config{}
	return config, json.Unmarshal([]byte(cm.Data["config"]), cm)
}
