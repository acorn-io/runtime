package strategy

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/storage"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func ToListOpts(namespace string, opts storage.ListOptions) *kclient.ListOptions {
	return &kclient.ListOptions{
		LabelSelector: opts.Predicate.Label,
		FieldSelector: opts.Predicate.Field,
		Namespace:     namespace,
		Limit:         opts.Predicate.Limit,
		Continue:      opts.Predicate.Continue,
		Raw: &metav1.ListOptions{
			ResourceVersion:      opts.ResourceVersion,
			ResourceVersionMatch: opts.ResourceVersionMatch,
			AllowWatchBookmarks:  opts.ProgressNotify || opts.Predicate.AllowWatchBookmarks,
		},
	}
}
