package imagemetadatacache

import (
	"context"
	"time"

	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var interval = 24 * time.Hour

func Purge(ctx context.Context, c kclient.Client) {
	for {
		err := doPurge(ctx, c)
		if err != nil {
			logrus.Errorf("Failed to purge imagemetadatacache: %v", err)
		}
		time.Sleep(interval)
	}
}

func doPurge(ctx context.Context, c kclient.Client) error {
	// We don't want to cache any of this data, so do live API calls
	list := &v1.PartialObjectMetadataList{
		TypeMeta: v1.TypeMeta{
			Kind:       "ImageMetadataCacheList",
			APIVersion: internalv1.SchemeGroupVersion.String(),
		},
	}

	if err := c.List(ctx, list); err != nil {
		return err
	}

	purge := time.Now().Add(-interval)
	for _, item := range list.Items {
		if item.CreationTimestamp.Time.Before(purge) {
			_ = c.Delete(ctx, &item)
		}
	}

	return nil
}
