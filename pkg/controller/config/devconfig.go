package config

import (
	"time"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func PurgeDevConfig(req router.Request, resp router.Response) error {
	cm := req.Object.(*corev1.ConfigMap)
	ttlString := cm.Annotations[labels.DevDeleteAfter]

	if ttlString != "" {
		expiration, err := time.Parse(time.RFC3339, ttlString)
		if err == nil {
			if time.Now().Before(expiration) {
				// Look 15 minutes later (stupid simple logic)
				resp.RetryAfter(15 * time.Minute)
				return nil
			}
			logrus.Infof("Time on dev config has expired [%s]", ttlString)
		} else {
			logrus.Errorf("failed to parse %s label value [%s], will delete config: %v", labels.DevDeleteAfter, ttlString, err)
		}
	}

	if credName := cm.Annotations[labels.DevCredentialName]; credName != "" {
		logrus.Infof("Deleting dev credential %s/%s", system.Namespace, credName)
		err := req.Client.Delete(req.Ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      credName,
				Namespace: system.Namespace,
			},
		})
		if err != nil && !apierror.IsNotFound(err) {
			return err
		}
	}

	logrus.Infof("Deleting dev config %s/%s", cm.Namespace, cm.Name)
	return req.Client.Delete(req.Ctx, cm)
}
