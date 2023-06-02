package scheme

import (
	acornadminapiv1 "github.com/acorn-io/acorn/pkg/apis/admin.acorn.io/v1"
	acornapiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	acornv1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	acornadminv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/rancher/wrangler/pkg/merr"
	"github.com/rancher/wrangler/pkg/schemes"
	appsv1 "k8s.io/api/apps/v1"
	authv1 "k8s.io/api/authorization/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

var (
	Scheme         = runtime.NewScheme()
	Codecs         = serializer.NewCodecFactory(Scheme)
	ParameterCodec = runtime.NewParameterCodec(Scheme)
)

func AddToScheme(scheme *runtime.Scheme) error {
	var errs []error
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
	errs = append(errs, acornv1.AddToScheme(scheme))
	errs = append(errs, acornapiv1.AddToScheme(scheme))
	errs = append(errs, acornadminv1.AddToScheme(scheme))
	errs = append(errs, acornadminapiv1.AddToScheme(scheme))
	errs = append(errs, corev1.AddToScheme(scheme))
	errs = append(errs, appsv1.AddToScheme(scheme))
	errs = append(errs, policyv1.AddToScheme(scheme))
	errs = append(errs, batchv1.AddToScheme(scheme))
	errs = append(errs, networkingv1.AddToScheme(scheme))
	errs = append(errs, storagev1.AddToScheme(scheme))
	errs = append(errs, apiregistrationv1.AddToScheme(scheme))
	errs = append(errs, rbacv1.AddToScheme(scheme))
	errs = append(errs, authv1.AddToScheme(scheme))
	errs = append(errs, apiextensionv1.AddToScheme(scheme))
	errs = append(errs, discoveryv1.AddToScheme(scheme))
	return merr.NewErrors(errs...)
}

func init() {
	utilruntime.Must(schemes.Register(AddToScheme))
	utilruntime.Must(AddToScheme(Scheme))
}
