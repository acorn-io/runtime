package watcher

import (
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ListOptions(ns string, options *internalversion.ListOptions) *client.ListOptions {
	return &client.ListOptions{
		LabelSelector: options.LabelSelector,
		FieldSelector: options.FieldSelector,
		Namespace:     ns,
		Limit:         options.Limit,
		Continue:      options.Continue,
		Raw: &metav1.ListOptions{
			LabelSelector:        options.LabelSelector.String(),
			FieldSelector:        options.FieldSelector.String(),
			Watch:                options.Watch,
			AllowWatchBookmarks:  options.AllowWatchBookmarks,
			ResourceVersion:      options.ResourceVersion,
			ResourceVersionMatch: options.ResourceVersionMatch,
			TimeoutSeconds:       options.TimeoutSeconds,
			Limit:                options.Limit,
			Continue:             options.Continue,
		},
	}
}
