package build

import (
	"context"
	"fmt"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/google/go-containerregistry/pkg/name"
	ggcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func digestOnlyImages(data map[string]v1.ImageData) (map[string]v1.ImageData, error) {
	result := map[string]v1.ImageData{}
	for k, v := range data {
		t, err := name.NewDigest(v.Image)
		if err != nil {
			return result, fmt.Errorf("parsing %s: %w", v.Image, err)
		}

		result[k] = v1.ImageData{
			Image: t.DigestStr(),
		}
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func digestOnlyContainers(data map[string]v1.ContainerData) (map[string]v1.ContainerData, error) {
	result := map[string]v1.ContainerData{}
	for k, v := range data {
		t, err := name.NewDigest(v.Image)
		if err != nil {
			return result, fmt.Errorf("parsing %s: %w", v.Image, err)
		}
		sidecars, err := digestOnlyImages(v.Sidecars)
		if err != nil {
			return nil, err
		}
		result[k] = v1.ContainerData{
			Image:    t.DigestStr(),
			Sidecars: sidecars,
		}
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func images(data map[string]v1.ImageData, opts []remote.Option) (result []ggcrv1.ImageIndex, _ error) {
	for _, entry := range typed.Sorted(data) {
		d, err := name.NewDigest(entry.Value.Image)
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

func containerImages(data map[string]v1.ContainerData, opts []remote.Option) (result []ggcrv1.ImageIndex, _ error) {
	for _, entry := range typed.Sorted(data) {
		d, err := name.NewDigest(entry.Value.Image)
		if err != nil {
			return nil, err
		}
		img, err := remote.Index(d, opts...)
		if err != nil {
			return nil, err
		}
		result = append(result, img)

		sidecarImages, err := images(entry.Value.Sidecars, opts)
		if err != nil {
			return nil, err
		}

		result = append(result, sidecarImages...)
	}

	return
}

func digestOnly(imageData v1.ImagesData) (result v1.ImagesData, err error) {
	result.Containers, err = digestOnlyContainers(imageData.Containers)
	if err != nil {
		return
	}

	result.Jobs, err = digestOnlyContainers(imageData.Jobs)
	if err != nil {
		return
	}

	result.Images, err = digestOnlyImages(imageData.Images)
	if err != nil {
		return
	}

	result.Acorns, err = digestOnlyImages(imageData.Acorns)
	return
}

func allImages(data v1.ImagesData, opts []remote.Option) (result []ggcrv1.ImageIndex, _ error) {
	remoteImages, err := containerImages(data.Containers, opts)
	if err != nil {
		return nil, err
	}
	result = append(result, remoteImages...)

	remoteImages, err = containerImages(data.Jobs, opts)
	if err != nil {
		return nil, err
	}
	result = append(result, remoteImages...)

	remoteImages, err = images(data.Images, opts)
	if err != nil {
		return nil, err
	}
	result = append(result, remoteImages...)

	remoteImages, err = images(data.Acorns, opts)
	if err != nil {
		return nil, err
	}
	result = append(result, remoteImages...)

	return
}

func createAppManifest(ctx context.Context, ref string, data v1.ImagesData, fullDigest bool) (string, error) {
	d, err := name.NewDigest(ref)
	if err != nil {
		return "", err
	}

	opts, err := GetRemoteOptions(ctx)
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

func createManifest(ctx context.Context, tags []string, platforms []v1.Platform) (string, error) {
	opts, err := GetRemoteOptions(ctx)
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
