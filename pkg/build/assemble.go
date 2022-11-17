package build

import (
	"context"
	"fmt"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/remoteopts"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/google/go-containerregistry/pkg/name"
	ggcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func images(data v1.ImagesData, opts []remote.Option) (result []ggcrv1.ImageIndex, _ error) {
	for _, entry := range typed.Sorted(data) {
		d, err := name.NewDigest(entry.Value)
		if err != nil {
			return nil, err
		}
		img, err := remote.Index(d, opts...)
		if err != nil {
			return nil, err
		}
		result = append(result, img)
	}
	return
}

func containerImages(data v1.ImagesData, opts []remote.Option) (result []ggcrv1.ImageIndex, _ error) {
	return images(data, opts)
}

func digestOnly(imageData v1.ImagesData) (v1.ImagesData, error) {
	result := v1.ImagesData{}
	for k, v := range imageData {
		t, err := name.NewDigest(v)
		if err != nil {
			return result, fmt.Errorf("parsing %s: %w", v, err)
		}

		result[k] = t.DigestStr()
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func allImages(data v1.ImagesData, opts []remote.Option) ([]ggcrv1.ImageIndex, error) {
	return containerImages(data, opts)
}

func createAppManifest(ctx context.Context, c client.Client, ref string, data v1.ImagesData, fullDigest bool) (string, error) {
	d, err := name.NewDigest(ref)
	if err != nil {
		return "", err
	}

	opts, err := remoteopts.WithClientDialer(ctx, c)
	if err != nil {
		return "", err
	}

	appImage, err := remote.Image(d, opts...)
	if err != nil {
		return "", err
	}

	index := mutate.AppendManifests(empty.Index, mutate.IndexAddendum{
		Add: appImage,
	})

	images, err := allImages(data, opts)
	if err != nil {
		return "", err
	}

	for _, image := range images {
		index = mutate.AppendManifests(index, mutate.IndexAddendum{
			Add: image,
		})
	}

	h, err := index.Digest()
	if err != nil {
		return "", err
	}

	err = remote.WriteIndex(d.Tag(h.Hex), index, opts...)
	if err != nil {
		return "", err
	}

	if fullDigest {
		return d.Digest(h.String()).Name(), nil
	}

	return h.Hex, nil
}

func createManifest(ctx context.Context, c client.Client, tags []string, platforms []v1.Platform) (string, error) {
	opts, err := remoteopts.WithClientDialer(ctx, c)
	if err != nil {
		return "", err
	}

	var (
		currentIndex = ggcrv1.ImageIndex(empty.Index)
		d            name.Digest
	)

	for i, tag := range tags {
		d, err = name.NewDigest(tag)
		if err != nil {
			return "", err
		}

		img, err := remote.Image(d, opts...)
		if err != nil {
			return "", err
		}

		platform := platforms[i]
		currentIndex = mutate.AppendManifests(currentIndex, mutate.IndexAddendum{
			Add: img,
			Descriptor: ggcrv1.Descriptor{
				Platform: &ggcrv1.Platform{
					Architecture: platform.Architecture,
					OS:           platform.OS,
					OSVersion:    platform.OSVersion,
					OSFeatures:   platform.OSFeatures,
					Variant:      platform.Variant,
				},
			},
		})
	}

	hash, err := currentIndex.Digest()
	if err != nil {
		return "", err
	}

	d = d.Digest(hash.String())
	err = remote.WriteIndex(d, currentIndex, opts...)
	return d.Name(), err
}
