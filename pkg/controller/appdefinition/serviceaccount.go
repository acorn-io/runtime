package appdefinition

import (
	"encoding/json"
	"fmt"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/awspermissions"
	"github.com/acorn-io/runtime/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// toServiceAccount generates a ServiceAccount for the container that includes annotations for AWS permissions if required
func toServiceAccount(req router.Request, saName string, labelMap, annotations map[string]string, appInstance *v1.AppInstance, perms v1.Permissions) (result kclient.Object, _ error) {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        saName,
			Namespace:   appInstance.Status.Namespace,
			Labels:      labelMap,
			Annotations: annotations,
		},
	}
	return sa, addAWS(req, appInstance, sa, perms)
}

func addAWS(req router.Request, appInstance *v1.AppInstance, sa *corev1.ServiceAccount, perms v1.Permissions) error {
	annotations, err := awspermissions.AWSAnnotations(req.Ctx, req.Client, appInstance, perms, sa.Name)
	if err != nil {
		return err
	}

	if perms.HasRules() {
		data, err := json.Marshal(perms)
		if err != nil {
			return fmt.Errorf("marshaling permission rules: %v", err)
		}
		if annotations == nil {
			annotations = map[string]string{}
		}
		annotations[labels.AcornPermissions] = string(data)
	}

	sa.Annotations = labels.Merge(sa.Annotations, annotations)
	return nil
}
