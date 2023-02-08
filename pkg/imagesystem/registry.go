package imagesystem

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/google/go-containerregistry/pkg/name"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NormalizeServerAddress(address string) string {
	if address == "docker.io" || address == "registry-1.docker.io" {
		return "index.docker.io"
	}
	return address
}

func GetInternalRepoForNamespace(ctx context.Context, c client.Reader, namespace string) (name.Repository, bool, error) {
	cfg, err := config.Get(ctx, c)
	if err != nil {
		return name.Repository{}, false, err
	}
	if *cfg.InternalRegistryPrefix != "" {
		n, err := name.NewRepository(*cfg.InternalRegistryPrefix + namespace)
		return n, true, err
	}

	dns, err := GetClusterInternalRegistryDNSName(ctx, c)
	if err != nil {
		return name.Repository{}, false, err
	}

	n, err := name.NewRepository(fmt.Sprintf("%s:%d/acorn/%s", dns, system.RegistryPort, namespace))
	return n, false, err
}

func GetRuntimePullableInternalRepoForNamespace(ctx context.Context, c client.Reader, namespace string) (name.Repository, error) {
	cfg, err := config.Get(ctx, c)
	if err != nil {
		return name.Repository{}, err
	}
	if *cfg.InternalRegistryPrefix != "" {
		return name.NewRepository(*cfg.InternalRegistryPrefix + namespace)
	}

	address, err := GetClusterInternalRegistryAddress(ctx, c)
	if err != nil {
		return name.Repository{}, err
	}

	return name.NewRepository(fmt.Sprintf("%s/acorn/%s", address, namespace))
}

func GetRuntimePullableInternalRepoForNamespaceAndID(ctx context.Context, c client.Reader, namespace, imageID string) (name.Reference, error) {
	var (
		repo name.Repository
	)
	image := &v1.ImageInstance{}
	if err := c.Get(ctx, router.Key(namespace, imageID), image); err == nil && image.Repo != "" {
		repo, err = name.NewRepository(image.Repo)
		if err != nil {
			return nil, err
		}
	} else if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	} else {
		repo, err = GetRuntimePullableInternalRepoForNamespace(ctx, c, namespace)
		if err != nil {
			return nil, err
		}
	}
	return repo.Digest("sha256:" + imageID), nil
}

func GetInternalRepoForNamespaceAndID(ctx context.Context, c client.Reader, namespace, imageID string) (name.Reference, error) {
	var (
		repo name.Repository
	)
	image := &v1.ImageInstance{}
	if err := c.Get(ctx, router.Key(namespace, imageID), image); err == nil && image.Repo != "" {
		repo, err = name.NewRepository(image.Repo)
		if err != nil {
			return nil, err
		}
	} else if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	} else {
		repo, _, err = GetInternalRepoForNamespace(ctx, c, namespace)
		if err != nil {
			return nil, err
		}
	}
	return repo.Digest("sha256:" + imageID), nil
}

func GetRegistryObjects(ctx context.Context, c client.Reader) (result []client.Object, _ error) {
	cfg, err := config.Get(ctx, c)
	if err != nil {
		return nil, err
	}
	if *cfg.InternalRegistryPrefix != "" {
		return nil, nil
	}

	result = append(result, registryService(system.ImagesNamespace)...)

	// we won't be able to find this service at first, so ignore the 404s
	port, err := getRegistryPort(ctx, c)
	if err == nil {
		result = append(result, containerdConfigPathDaemonSet(system.ImagesNamespace, system.DefaultImage(), strconv.Itoa(port))...)
	} else if !apierrors.IsNotFound(err) {
		return nil, err
	}

	result = append(result, registryDeployment(system.ImagesNamespace, system.DefaultImage())...)
	return result, nil
}

func GetClusterInternalRegistryDNSName(ctx context.Context, c client.Reader) (string, error) {
	cfg, err := config.Get(ctx, c)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s.%s", system.RegistryName, system.ImagesNamespace, cfg.InternalClusterDomain), err
}

func IsClusterInternalRegistryAddressReference(url string) bool {
	return strings.HasPrefix(url, "127.")
}

func GetClusterInternalRegistryAddress(ctx context.Context, c client.Reader) (string, error) {
	port, err := getRegistryPort(ctx, c)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("127.0.0.1:%d", port), nil
}

func getRegistryPort(ctx context.Context, c client.Reader) (int, error) {
	var service corev1.Service
	err := c.Get(ctx, client.ObjectKey{Name: system.RegistryName, Namespace: system.ImagesNamespace}, &service)
	if err != nil {
		return 0, fmt.Errorf("getting %s/%s service: %w", system.ImagesNamespace, system.RegistryName, err)
	}
	for _, port := range service.Spec.Ports {
		if port.Name == system.RegistryName && port.NodePort > 0 {
			return int(port.NodePort), nil
		}
	}

	return 0, fmt.Errorf("failed to find node port for registry %s/%s", system.ImagesNamespace, system.RegistryName)
}

func IsNotInternalRepo(ctx context.Context, c client.Reader, image string) error {
	if !strings.Contains(image, "/") {
		return nil
	}

	cfg, err := config.Get(ctx, c)
	if err != nil {
		return err
	}

	return isNotInternalRepo(*cfg.InternalRegistryPrefix, image)
}

func isNotInternalRepo(prefix, image string) error {
	if os.Getenv("ACORN_TEST_ALLOW_LOCALHOST_REGISTRY") != "true" && IsClusterInternalRegistryAddressReference(image) {
		return fmt.Errorf("invalid image reference %s", image)
	}

	if prefix == "" {
		return nil
	}

	if strings.HasPrefix(image, prefix) {
		return fmt.Errorf("invalid image reference prefix %s", image)
	}

	imageHostPort, _, _ := strings.Cut(image, "/")
	imageHost, _, _ := strings.Cut(imageHostPort, ":")
	prefixHostPort, _, _ := strings.Cut(prefix, "/")
	prefixHost, _, _ := strings.Cut(prefixHostPort, ":")
	newImage := strings.Replace(image, imageHostPort, NormalizeServerAddress(imageHost), 1)
	newPrefix := strings.Replace(prefix, prefixHostPort, NormalizeServerAddress(prefixHost), 1)
	if strings.HasPrefix(newImage, newPrefix) {
		return fmt.Errorf("invalid image reference prefix %s", image)
	}

	return nil
}

func ParseAndEnsureNotInternalRepo(ctx context.Context, c client.Reader, image string) (name.Reference, error) {
	if err := IsNotInternalRepo(ctx, c, image); err != nil {
		return nil, err
	}
	return name.ParseReference(image)
}
