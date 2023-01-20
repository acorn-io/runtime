package build

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/google/go-containerregistry/pkg/name"
	ggcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/sirupsen/logrus"
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

func digestToIndexAddendum(ref string, opts []remote.Option) (*mutate.IndexAddendum, error) {
	d, err := name.NewDigest(ref)
	if err != nil {
		return nil, err
	}
	descriptor, err := remote.Head(d, opts...)
	if err != nil {
		return nil, err
	}

	if descriptor.MediaType.IsIndex() {
		img, err := remote.Index(d, opts...)
		if err != nil {
			return nil, err
		}
		return &mutate.IndexAddendum{
			Add: img,
		}, nil
	}

	img, err := remote.Image(d, opts...)
	if err != nil {
		return nil, err
	}

	platform, err := imagePlatform(img)
	if err != nil {
		return nil, err
	}

	return &mutate.IndexAddendum{
		Add: img,
		Descriptor: ggcrv1.Descriptor{
			Platform: platform,
		},
	}, nil
}

func images(data map[string]v1.ImageData, opts []remote.Option) (result []mutate.IndexAddendum, _ error) {
	for _, entry := range typed.Sorted(data) {
		add, err := digestToIndexAddendum(entry.Value.Image, opts)
		if err != nil {
			return nil, err
		}
		result = append(result, *add)
	}
	return
}

func containerImages(data map[string]v1.ContainerData, opts []remote.Option) (result []mutate.IndexAddendum, _ error) {
	for _, entry := range typed.Sorted(data) {
		add, err := digestToIndexAddendum(entry.Value.Image, opts)
		if err != nil {
			return nil, err
		}
		result = append(result, *add)

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
	return
}

func allImages(data v1.ImagesData, opts []remote.Option) (result []mutate.IndexAddendum, _ error) {
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

	return
}

func imagePlatform(img ggcrv1.Image) (*ggcrv1.Platform, error) {
	config, err := img.ConfigFile()
	if err != nil {
		return nil, err
	}
	return &ggcrv1.Platform{
		Architecture: config.Architecture,
		OS:           config.OS,
		OSVersion:    config.OSVersion,
		Variant:      config.Variant,
	}, nil
}

// retryGet will keep trying to get a digest for 5 seconds until it succeeds. This is specifically used for digests
// we just created. For example, in ECR this call will sometimes return 404 while I assume S3 is becoming eventually
// consistent. It is possible that you do GET and find the response and then do GET and get a 404. So just
// keep trying until we get it.
func retryGetImage(d name.Digest, opts []remote.Option) (result ggcrv1.Image, err error) {
	for i := 0; i < 5; i++ {
		result, err = remote.Image(d, opts...)
		if err == nil {
			return
		} else {
			logrus.Warnf("failed to find newly created manifest %s, retrying: %v", d.String(), err)
		}
		time.Sleep(time.Second)
	}

	return
}

func createAppManifest(ctx context.Context, ref string, data v1.ImagesData, fullDigest bool, opts []remote.Option) (string, error) {
	d, err := name.NewDigest(ref)
	if err != nil {
		return "", err
	}

	appImage, err := retryGetImage(d, opts)
	if err != nil {
		return "", fmt.Errorf("failed to find app metadata image: %w", err)
	}

	platform, err := imagePlatform(appImage)
	if err != nil {
		return "", err
	}

	index := mutate.AppendManifests(mutate.IndexMediaType(empty.Index, types.DockerManifestList), mutate.IndexAddendum{
		Add: appImage,
		Descriptor: ggcrv1.Descriptor{
			Platform: platform,
		},
	})

	images, err := allImages(data, opts)
	if err != nil {
		return "", err
	}

	index = mutate.AppendManifests(index, images...)

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

func createManifest(tags []string, platforms []v1.Platform, opts []remote.Option) (string, error) {
	var (
		currentIndex = ggcrv1.ImageIndex(empty.Index)
		d            name.Digest
		err          error
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
