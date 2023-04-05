package autoupgrade

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockDaemonClient struct {
	apps                                []v1.AppInstance
	defaultAutoUpgradeInterval          string
	appUpdates                          map[string]string
	localTags, remoteTags               []string
	remoteImageDigest, resolvedLocalTag string
	localTagFound                       bool
	imageDenyList                       map[string]struct{}
}

func (m *mockDaemonClient) getConfig(_ context.Context) (*apiv1.Config, error) {
	return &apiv1.Config{AutoUpgradeInterval: &m.defaultAutoUpgradeInterval}, nil
}

func (m *mockDaemonClient) listAppInstances(_ context.Context) ([]v1.AppInstance, error) {
	return m.apps, nil
}

func (m *mockDaemonClient) updateAppStatus(_ context.Context, instance *v1.AppInstance) error {
	if m.appUpdates == nil {
		m.appUpdates = make(map[string]string)
	}
	// Use the concatenation of these values because one will be empty and the other will have the new image.
	m.appUpdates[instance.Name] = instance.Status.AvailableAppImage + instance.Status.ConfirmUpgradeAppImage
	return nil
}

func (m *mockDaemonClient) listTags(context.Context, string, string, ...remote.Option) ([]string, error) {
	return m.remoteTags, nil
}

func (m *mockDaemonClient) getTagsMatchingRepo(context.Context, name.Reference, string, string) ([]string, error) {
	return m.localTags, nil
}

func (m *mockDaemonClient) imageDigest(_ context.Context, namespace, name string, _ ...remote.Option) (string, error) {
	if m.remoteImageDigest != "" {
		return m.remoteImageDigest, nil
	}
	return fmt.Sprintf("sha256:%s1234%sabcd", namespace, name), nil
}

func (m *mockDaemonClient) resolveLocalTag(context.Context, string, string) (string, bool, error) {
	return m.resolvedLocalTag, m.localTagFound, nil
}

func (m *mockDaemonClient) checkImageAllowed(_ context.Context, _ string, img string) error {
	if _, ok := m.imageDenyList[img]; ok {
		return fmt.Errorf("Mock error - Image %s Denied", img)
	}
	return nil
}

func TestDetermineAppsToRefresh(t *testing.T) {
	defaultNextCheckInterval := time.Minute
	now := time.Now()
	thirtySecondsAgo := now.Add(-30 * time.Second)
	oneMinuteAgo := now.Add(-time.Minute)
	ptrTrue := &[]bool{true}[0]
	appImages := map[string]string{
		"test-1":          "acorn/test-1:v#.*.**",
		"acorn-1":         "acorn/acorn-1:v1.1.1-*",
		"test-2":          "other/test-2:v1.2.3-**",
		"no-auto-upgrade": "acorn/no-auto-upgrade:latest",
		"brand-new":       "acorn/brand-new:v#.#.#",
		"other-test-1":    "acorn/test-1:v#.#.#",
		"enabled-app":     "acorn/enabled:latest",
		"notify-app":      "acorn/notify:latest",
	}
	apps := make(map[kclient.ObjectKey]v1.AppInstance, len(appImages))
	for _, entry := range typed.Sorted(appImages) {
		app := v1.AppInstance{
			ObjectMeta: metav1.ObjectMeta{Name: entry.Key, Namespace: "acorn"},
			Spec:       v1.AppInstanceSpec{Image: entry.Value},
		}
		switch entry.Key {
		case "brand-new":
			app.CreationTimestamp = metav1.Now()
		case "enabled-app":
			app.Spec.AutoUpgrade = ptrTrue
		case "notify-app":
			app.Spec.NotifyUpgrade = ptrTrue
		}
		apps[router.Key(app.Namespace, app.Name)] = app
	}

	tests := []struct {
		name                                          string
		appKeysPrevCheckBefore, appKeysPrevCheckAfter map[kclient.ObjectKey]time.Time
		want                                          map[imageAndNamespaceKey][]kclient.ObjectKey
	}{
		{
			name:                   "No auto-upgrade apps",
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{},
			want:                   make(map[imageAndNamespaceKey][]kclient.ObjectKey),
		},
		{
			name:                   "App doesn't exist",
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "acorn"): oneMinuteAgo},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{},
			want:                   make(map[imageAndNamespaceKey][]kclient.ObjectKey),
		},
		{
			name:                   "Auto upgrade was turned off",
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "no-auto-upgrade"): oneMinuteAgo},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{},
			want:                   make(map[imageAndNamespaceKey][]kclient.ObjectKey),
		},
		{
			name:                   "Not time to check",
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "test-1"): thirtySecondsAgo},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "test-1"): thirtySecondsAgo},
			want:                   make(map[imageAndNamespaceKey][]kclient.ObjectKey),
		},
		{
			name:                   "Time to check",
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "test-1"): oneMinuteAgo},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "test-1"): oneMinuteAgo},
			want:                   map[imageAndNamespaceKey][]kclient.ObjectKey{{"acorn/test-1", "acorn"}: {router.Key("acorn", "test-1")}},
		},
		{
			name:                   "App that was deleted and recreated, creation timestamp newer than last upgrade time",
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "brand-new"): thirtySecondsAgo},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "brand-new"): thirtySecondsAgo},
			want:                   map[imageAndNamespaceKey][]kclient.ObjectKey{{"acorn/brand-new", "acorn"}: {router.Key("acorn", "brand-new")}},
		},
		{
			name:                   "Two apps using the same image need to be updated",
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "test-1"): oneMinuteAgo, router.Key("acorn", "other-test-1"): oneMinuteAgo},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "test-1"): oneMinuteAgo, router.Key("acorn", "other-test-1"): oneMinuteAgo},
			want:                   map[imageAndNamespaceKey][]kclient.ObjectKey{{"acorn/test-1", "acorn"}: {router.Key("acorn", "test-1"), router.Key("acorn", "other-test-1")}},
		},
		{
			name:                   "Auto upgrade enabled with no tag pattern",
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "enabled-app"): oneMinuteAgo},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "enabled-app"): oneMinuteAgo},
			want:                   map[imageAndNamespaceKey][]kclient.ObjectKey{{"acorn/enabled:latest", "acorn"}: {router.Key("acorn", "enabled-app")}},
		},
		{
			name:                   "Notify upgrade enabled with no tag pattern",
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "notify-app"): oneMinuteAgo},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "notify-app"): oneMinuteAgo},
			want:                   map[imageAndNamespaceKey][]kclient.ObjectKey{{"acorn/notify:latest", "acorn"}: {router.Key("acorn", "notify-app")}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &daemon{
				appKeysPrevCheck: tt.appKeysPrevCheckBefore,
			}

			got := d.determineAppsToRefresh(apps, defaultNextCheckInterval, time.Now())
			assert.Equalf(t, len(got), len(tt.want), "length of maps don't match")
			for key, value := range got {
				assert.ElementsMatchf(t, tt.want[key], value, "determineAppsToRefresh(%v, %v)", apps, defaultNextCheckInterval)
			}
			assert.Equalf(t, tt.appKeysPrevCheckAfter, d.appKeysPrevCheck, "daemon state not as expected after call")
		})
	}
}

func TestRefreshImages(t *testing.T) {
	now := time.Now()
	thirtySecondsAgo := now.Add(-30 * time.Second)
	ptrTrue := &[]bool{true}[0]
	appImages := map[string]string{
		"test-1":      "acorn/test-1:v#.#.#",
		"acorn-1":     "docker.io/acorn/acorn-1:v1.1.1-*",
		"enabled-app": "docker.io/acorn/enabled:latest",
		"notify-app":  "docker.io/acorn/notify:latest",
	}
	apps := make(map[kclient.ObjectKey]v1.AppInstance, len(appImages))
	for _, entry := range typed.Sorted(appImages) {
		app := v1.AppInstance{
			ObjectMeta: metav1.ObjectMeta{Name: entry.Key, Namespace: "acorn"},
			Spec:       v1.AppInstanceSpec{Image: entry.Value},
			Status:     v1.AppInstanceStatus{AppImage: v1.AppImage{Digest: fmt.Sprintf("sha256:acorn1234%sabcd", strings.Split(entry.Value, ":")[0])}},
		}
		switch entry.Key {
		case "enabled-app":
			app.Spec.AutoUpgrade = ptrTrue
		case "notify-app":
			app.Spec.NotifyUpgrade = ptrTrue
		}
		apps[router.Key(app.Namespace, app.Name)] = app
	}

	tests := []struct {
		name                                          string
		client                                        *mockDaemonClient
		appKeysPrevCheckBefore, appKeysPrevCheckAfter map[kclient.ObjectKey]time.Time
		imagesToRefresh                               map[imageAndNamespaceKey][]kclient.ObjectKey
		appsUpdated                                   map[string]string
	}{
		{
			name:   "No images to refresh",
			client: &mockDaemonClient{},
		},
		{
			name:                   "Auto refresh tag with local update",
			client:                 &mockDaemonClient{localTags: []string{"v1.1.1"}},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "test-1"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "acorn/test-1", namespace: "acorn"}: {router.Key("acorn", "test-1")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "test-1"): now},
			appsUpdated:            map[string]string{"test-1": "acorn/test-1:v1.1.1"},
		},
		{
			name:                   "Auto refresh tag with multiple local updates",
			client:                 &mockDaemonClient{localTags: []string{"v1.1.1", "v1.1.2"}},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "test-1"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "acorn/test-1", namespace: "acorn"}: {router.Key("acorn", "test-1")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "test-1"): now},
			appsUpdated:            map[string]string{"test-1": "acorn/test-1:v1.1.2"},
		},
		{
			name:                   "Auto refresh tag with remote update",
			client:                 &mockDaemonClient{remoteTags: []string{"v1.1.1-alpha"}},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "acorn-1"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "docker.io/acorn/acorn-1", namespace: "acorn"}: {router.Key("acorn", "acorn-1")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "acorn-1"): now},
			appsUpdated:            map[string]string{"acorn-1": "index.docker.io/acorn/acorn-1:v1.1.1-alpha"},
		},
		{
			name:                   "Auto refresh tag with multiple remote updates",
			client:                 &mockDaemonClient{remoteTags: []string{"v1.1.1-alpha", "v1.1.1-beta"}},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "acorn-1"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "docker.io/acorn/acorn-1", namespace: "acorn"}: {router.Key("acorn", "acorn-1")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "acorn-1"): now},
			appsUpdated:            map[string]string{"acorn-1": "index.docker.io/acorn/acorn-1:v1.1.1-beta"},
		},
		{
			name:                   "Auto refresh tag with local and remote updates, local wins",
			client:                 &mockDaemonClient{localTags: []string{"v1.1.1-beta"}, remoteTags: []string{"v1.1.1-alpha"}},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "acorn-1"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "docker.io/acorn/acorn-1", namespace: "acorn"}: {router.Key("acorn", "acorn-1")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "acorn-1"): now},
			appsUpdated:            map[string]string{"acorn-1": "index.docker.io/acorn/acorn-1:v1.1.1-beta"},
		},
		{
			name:                   "Auto refresh tag with local and remote updates, remote wins",
			client:                 &mockDaemonClient{localTags: []string{"v1.1.1-alpha"}, remoteTags: []string{"v1.1.1-beta"}},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "acorn-1"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "docker.io/acorn/acorn-1", namespace: "acorn"}: {router.Key("acorn", "acorn-1")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "acorn-1"): now},
			appsUpdated:            map[string]string{"acorn-1": "index.docker.io/acorn/acorn-1:v1.1.1-beta"},
		},
		{
			name:                   "Auto refresh tag with local and remote tags, no updates",
			client:                 &mockDaemonClient{localTags: []string{"v1.1.2-alpha"}, remoteTags: []string{"v1.1.2-beta"}},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "acorn-1"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "docker.io/acorn/acorn-1", namespace: "acorn"}: {router.Key("acorn", "acorn-1")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "acorn-1"): now},
		},
		{
			name:                   "Auto refresh tag with remote digest change",
			client:                 &mockDaemonClient{remoteImageDigest: "sha256:acorn4321docker.io/acorn/acorn-1dcba"},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "acorn-1"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "docker.io/acorn/acorn-1", namespace: "acorn"}: {router.Key("acorn", "acorn-1")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "acorn-1"): now},
			appsUpdated:            map[string]string{"acorn-1": "docker.io/acorn/acorn-1"},
		},
		{
			name:                   "Auto refresh tag with local digest change",
			client:                 &mockDaemonClient{resolvedLocalTag: "sha256:acorn4321docker.io/acorn/acorn-1dcba", localTagFound: true},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "acorn-1"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "docker.io/acorn/acorn-1", namespace: "acorn"}: {router.Key("acorn", "acorn-1")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "acorn-1"): now},
			appsUpdated:            map[string]string{"acorn-1": "docker.io/acorn/acorn-1"},
		},
		{
			name:                   "Auto refresh tag with no local digest found",
			client:                 &mockDaemonClient{resolvedLocalTag: "sha256:acorn4321docker.io/acorn/acorn-1dcba"},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "acorn-1"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "docker.io/acorn/acorn-1", namespace: "acorn"}: {router.Key("acorn", "acorn-1")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "acorn-1"): now},
		},
		{
			name:                   "Auto refresh enabled with local digest change",
			client:                 &mockDaemonClient{resolvedLocalTag: "sha256:acorn4321docker.io/acorn/enableddcba", localTagFound: true},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "enabled-app"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "docker.io/acorn/enabled", namespace: "acorn"}: {router.Key("acorn", "enabled-app")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "enabled-app"): now},
			appsUpdated:            map[string]string{"enabled-app": "docker.io/acorn/enabled"},
		},
		{
			name:                   "Auto refresh enabled with no local digest change",
			client:                 &mockDaemonClient{localTagFound: true},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "enabled-app"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "docker.io/acorn/enabled", namespace: "acorn"}: {router.Key("acorn", "enabled-app")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "enabled-app"): now},
		},
		{
			name:                   "Auto refresh enabled with remote digest change, no local",
			client:                 &mockDaemonClient{remoteImageDigest: "sha256:acorn4321docker.io/acorn/enableddcba"},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "enabled-app"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "docker.io/acorn/enabled", namespace: "acorn"}: {router.Key("acorn", "enabled-app")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "enabled-app"): now},
			appsUpdated:            map[string]string{"enabled-app": "docker.io/acorn/enabled"},
		},
		{
			name:                   "Auto refresh notify with local digest change",
			client:                 &mockDaemonClient{resolvedLocalTag: "sha256:acorn4321docker.io/acorn/notifydcba", localTagFound: true},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "notify-app"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "docker.io/acorn/notify", namespace: "acorn"}: {router.Key("acorn", "notify-app")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "notify-app"): now},
			appsUpdated:            map[string]string{"notify-app": "docker.io/acorn/notify"},
		},
		{
			name:                   "Auto refresh notify with no local digest change",
			client:                 &mockDaemonClient{localTagFound: true},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "notify-app"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "docker.io/acorn/notify", namespace: "acorn"}: {router.Key("acorn", "notify-app")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "notify-app"): now},
		},
		{
			name:                   "Auto refresh notify with remote digest change, no local",
			client:                 &mockDaemonClient{remoteImageDigest: "sha256:acorn4321docker.io/acorn/notifydcba"},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "notify-app"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "docker.io/acorn/notify", namespace: "acorn"}: {router.Key("acorn", "notify-app")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "notify-app"): now},
			appsUpdated:            map[string]string{"notify-app": "docker.io/acorn/notify"},
		},
		{
			name:                   "Auto refresh tag with multiple remote tags with latest tag image denied and current being an older image",
			client:                 &mockDaemonClient{remoteTags: []string{"v1.1.1", "v1.1.2"}, imageDenyList: map[string]struct{}{"index.docker.io/acorn/test-1:v1.1.2": {}}},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "test-1"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "index.docker.io/acorn/test-1:v1.1.1", namespace: "acorn"}: {router.Key("acorn", "test-1")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "test-1"): now},
			appsUpdated:            map[string]string{"test-1": "index.docker.io/acorn/test-1:v1.1.1"},
		},
		{
			name:                   "Auto refresh tag with multiple remote tags with latest tag image denied and current being latest image",
			client:                 &mockDaemonClient{remoteTags: []string{"v1.1.1", "v1.1.2"}, imageDenyList: map[string]struct{}{"index.docker.io/acorn/test-1:v1.1.2": {}}},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "test-1"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "index.docker.io/acorn/test-1:v1.1.2", namespace: "acorn"}: {router.Key("acorn", "test-1")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "test-1"): now},
			appsUpdated:            map[string]string{"test-1": "index.docker.io/acorn/test-1:v1.1.2"},
		},
		{
			name:                   "Auto refresh tag with multiple remote tags with non-latest tag image denied and current being the oldest image",
			client:                 &mockDaemonClient{remoteTags: []string{"v1.1.1", "v1.1.2", "v1.1.3"}, imageDenyList: map[string]struct{}{"index.docker.io/acorn/test-1:v1.1.2": {}}},
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{router.Key("acorn", "test-1"): thirtySecondsAgo},
			imagesToRefresh:        map[imageAndNamespaceKey][]kclient.ObjectKey{{image: "index.docker.io/acorn/test-1", namespace: "acorn"}: {router.Key("acorn", "test-1")}},
			appKeysPrevCheckAfter:  map[kclient.ObjectKey]time.Time{router.Key("acorn", "test-1"): now},
			appsUpdated:            map[string]string{"test-1": "index.docker.io/acorn/test-1:v1.1.3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &daemon{
				client:           tt.client,
				appKeysPrevCheck: tt.appKeysPrevCheckBefore,
			}

			d.refreshImages(context.Background(), apps, tt.imagesToRefresh, now)
			assert.Equalf(t, tt.appKeysPrevCheckAfter, d.appKeysPrevCheck, "daemon state not as expected after call")

			assert.Equalf(t, len(tt.appsUpdated), len(tt.client.appUpdates), "different number of apps updated than expected")
			for appName, image := range tt.appsUpdated {
				assert.Equalf(t, image, tt.client.appUpdates[appName], "%s app doesn't have expected new version", appName)
			}
		})
	}
}
func TestDaemonSync(t *testing.T) {
	start := time.Now()
	tenMinutesAgo := time.Now().Add(-10 * time.Minute)
	fiftySecondsAgo := time.Now().Add(-50 * time.Second)
	ptrTrue := &[]bool{true}[0]
	appImages := map[string]string{
		"test-1":          "30s",
		"acorn-1":         "1m",
		"enabled-app":     "5m",
		"notify-app":      "3m",
		"bad-interval":    "help",
		"no-auto-upgrade": "1s",
	}
	apps := make([]v1.AppInstance, 0, len(appImages))
	for _, entry := range typed.Sorted(appImages) {
		app := v1.AppInstance{
			ObjectMeta: metav1.ObjectMeta{Name: entry.Key, Namespace: "acorn"},
			Spec:       v1.AppInstanceSpec{AutoUpgradeInterval: entry.Value},
		}
		switch entry.Key {
		case "enabled-app":
			app.Spec.AutoUpgrade = ptrTrue
			app.Spec.Image = "acorn/acorn-enabled:latest"
		case "notify-app":
			app.Spec.NotifyUpgrade = ptrTrue
			app.Spec.Image = "acorn/acorn-notify:latest"
		case "no-auto-upgrade":
			app.Spec.Image = "acorn/acorn:latest"
		default:
			app.Spec.Image = "acorn/acorn:v#.*.**"
		}
		apps = append(apps, app)
	}

	tests := []struct {
		name                                          string
		apps                                          []v1.AppInstance
		appKeysPrevCheckBefore, appKeysPrevCheckAfter map[kclient.ObjectKey]time.Time
		defaultUpgradeInterval                        string
		expectedNextCheckInterval                     time.Duration
	}{
		{
			name:                      "No images to refresh",
			defaultUpgradeInterval:    "1h",
			expectedNextCheckInterval: time.Hour,
		},
		{
			name:                   "All apps need refreshed because they are just added",
			apps:                   apps,
			defaultUpgradeInterval: "1h",
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{},
			appKeysPrevCheckAfter: map[kclient.ObjectKey]time.Time{
				router.Key("acorn", "test-1"):      start,
				router.Key("acorn", "acorn-1"):     start,
				router.Key("acorn", "enabled-app"): start,
				router.Key("acorn", "notify-app"):  start,
				// Not able to calculate refresh interval, so app is not updated.
				router.Key("acorn", "bad-interval"): {},
			},
			expectedNextCheckInterval: 30 * time.Second,
		},
		{
			name:                   "All apps need refreshed because they are just added, but default sync time is next refresh",
			apps:                   apps,
			defaultUpgradeInterval: "1s",
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{},
			appKeysPrevCheckAfter: map[kclient.ObjectKey]time.Time{
				router.Key("acorn", "test-1"):      start,
				router.Key("acorn", "acorn-1"):     start,
				router.Key("acorn", "enabled-app"): start,
				router.Key("acorn", "notify-app"):  start,
				// Not able to calculate refresh interval, so app is not updated.
				router.Key("acorn", "bad-interval"): {},
			},
			expectedNextCheckInterval: time.Second,
		},
		{
			name:                   "None of the apps need to be refreshed",
			apps:                   apps,
			defaultUpgradeInterval: "1h",
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{
				router.Key("acorn", "test-1"):      start,
				router.Key("acorn", "acorn-1"):     start,
				router.Key("acorn", "enabled-app"): start,
				router.Key("acorn", "notify-app"):  start,
			},
			appKeysPrevCheckAfter: map[kclient.ObjectKey]time.Time{
				router.Key("acorn", "test-1"):      start,
				router.Key("acorn", "acorn-1"):     start,
				router.Key("acorn", "enabled-app"): start,
				router.Key("acorn", "notify-app"):  start,
				// Not able to calculate refresh interval, so app is not updated.
				router.Key("acorn", "bad-interval"): {},
			},
			expectedNextCheckInterval: 30 * time.Second,
		},
		{
			name:                   "Apps need to be refreshed based on time",
			apps:                   apps,
			defaultUpgradeInterval: "1h",
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{
				router.Key("acorn", "test-1"):      tenMinutesAgo,
				router.Key("acorn", "acorn-1"):     tenMinutesAgo,
				router.Key("acorn", "enabled-app"): tenMinutesAgo,
				router.Key("acorn", "notify-app"):  tenMinutesAgo,
			},
			appKeysPrevCheckAfter: map[kclient.ObjectKey]time.Time{
				router.Key("acorn", "test-1"):      start,
				router.Key("acorn", "acorn-1"):     start,
				router.Key("acorn", "enabled-app"): start,
				router.Key("acorn", "notify-app"):  start,
				// Not able to calculate refresh interval, so app is not updated.
				router.Key("acorn", "bad-interval"): {},
			},
			expectedNextCheckInterval: 30 * time.Second,
		},
		{
			name:                   "Ensure a shorter next check is returned when an app is nearing update time",
			apps:                   apps,
			defaultUpgradeInterval: "1h",
			appKeysPrevCheckBefore: map[kclient.ObjectKey]time.Time{
				router.Key("acorn", "test-1"):      start,
				router.Key("acorn", "acorn-1"):     fiftySecondsAgo,
				router.Key("acorn", "enabled-app"): start,
				router.Key("acorn", "notify-app"):  start,
			},
			appKeysPrevCheckAfter: map[kclient.ObjectKey]time.Time{
				router.Key("acorn", "test-1"):      start,
				router.Key("acorn", "acorn-1"):     fiftySecondsAgo,
				router.Key("acorn", "enabled-app"): start,
				router.Key("acorn", "notify-app"):  start,
				// Not able to calculate refresh interval, so app is not updated.
				router.Key("acorn", "bad-interval"): {},
			},
			expectedNextCheckInterval: 10 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &daemon{
				client:           &mockDaemonClient{apps: tt.apps, defaultAutoUpgradeInterval: tt.defaultUpgradeInterval},
				appKeysPrevCheck: tt.appKeysPrevCheckBefore,
			}

			got, _ := d.sync(context.Background(), start)
			// Calculate diff between start and now because that is how far off we would expect the "actual" and "expected" to be.
			diff := time.Since(start)
			assert.Equalf(t, tt.appKeysPrevCheckAfter, d.appKeysPrevCheck, "daemon state not as expected after call")

			// The expected next check interval will be slightly smaller than expected because of the time it takes to run the test.
			// Assert that it is within the expected diff.
			assert.InDeltaf(t, tt.expectedNextCheckInterval, got, float64(diff), "Next check interval much different than expected")
		})
	}
}
