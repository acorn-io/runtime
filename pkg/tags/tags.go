package tags

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/moby/locker"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConfigMapName = "acorn-tags"
	ConfigMapKey  = "acorn-tags"
)

var (
	SHAPermissivePrefixPattern = regexp.MustCompile(`^[a-f\d]{3,64}$`)
	SHAPattern                 = regexp.MustCompile(`^[a-f\d]{64}$`)
	tagLock                    locker.Locker
)

func IsLocalReference(image string) bool {
	if strings.HasPrefix(image, "sha256:") {
		return true
	}
	if SHAPermissivePrefixPattern.MatchString(image) {
		return true
	}
	return false
}

func getConfigMap(ctx context.Context, c client.Reader, namespace string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	return configMap, c.Get(ctx, client.ObjectKey{
		Name:      ConfigMapName,
		Namespace: namespace,
	}, configMap)
}

func Remove(ctx context.Context, c client.Client, namespace, digest, tag string) (int, error) {
	tagLock.Lock(namespace)
	defer func() { _ = tagLock.Unlock(namespace) }()

	configMap, err := getConfigMap(ctx, c, namespace)
	if apierrors.IsNotFound(err) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	mapData := map[string][]string{}
	data := []byte(configMap.Data[ConfigMapKey])
	if len(data) == 0 {
		return 0, nil
	}

	if err := json.Unmarshal(data, &mapData); err != nil {
		return 0, err
	}

	key := strings.TrimPrefix(digest, "sha256:")
	var newTags []string
	for _, oldTag := range mapData[key] {
		if oldTag == tag {
			continue
		}
		newTags = append(newTags, oldTag)
	}

	if len(newTags) == 0 {
		delete(mapData, key)
	} else {
		mapData[key] = newTags
	}

	data, err = json.Marshal(mapData)
	if err != nil {
		return 0, err
	}

	configMap.Data[ConfigMapKey] = string(data)
	return len(newTags), c.Update(ctx, configMap)
}

func Write(ctx context.Context, c client.Client, namespace, digest, tag string) error {
	tagLock.Lock(namespace)
	defer func() { _ = tagLock.Unlock(namespace) }()

	configMap, err := getConfigMap(ctx, c, namespace)
	if apierrors.IsNotFound(err) {
		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ConfigMapName,
				Namespace: namespace,
			},
		}
	} else if err != nil {
		return err
	}

	mapData := map[string][]string{}
	data := []byte(configMap.Data[ConfigMapKey])
	if len(data) > 0 {
		if err := json.Unmarshal(data, &mapData); err != nil {
			return err
		}
	}

	key := strings.TrimPrefix(digest, "sha256:")
	if slices.Contains(mapData[key], tag) {
		return nil
	}

	normalizedTag, err := name.NewTag(tag)
	if err != nil {
		return err
	}

	for key, tags := range mapData {
		for i, existingTag := range tags {
			normalizedExistingTag, err := name.NewTag(existingTag)
			if err != nil || normalizedExistingTag.Name() == normalizedTag.Name() {
				mapData[key] = append(tags[:i], tags[i+1:]...)
				continue
			}
		}
	}

	mapData[key] = append(mapData[key], tag)
	data, err = json.Marshal(mapData)
	if err != nil {
		return err
	}

	if configMap.Data == nil {
		configMap.Data = map[string]string{}
	}

	configMap.Data[ConfigMapKey] = string(data)
	if configMap.UID == "" {
		createErr := c.Create(ctx, configMap)
		if apierrors.IsNotFound(createErr) {
			err = c.Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: configMap.Namespace,
				},
			})
			if err != nil {
				return createErr
			}
			return c.Create(ctx, configMap)
		} else {
			return createErr
		}
	}
	return c.Update(ctx, configMap)
}

func Get(ctx context.Context, c client.Reader, namespace string) (map[string][]string, error) {
	configMap, err := getConfigMap(ctx, c, namespace)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}

	data := configMap.Data[ConfigMapKey]
	if len(data) == 0 {
		return nil, nil
	}

	result := map[string][]string{}
	return result, json.Unmarshal([]byte(data), &result)
}
