package config

import (
	"context"
	"testing"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	mocks "github.com/acorn-io/acorn/pkg/mocks/k8s"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestAcornDNSDisabledNoLookupsHappen(t *testing.T) {
	s := "not exactly disabled, but any string that doesn't equal" +
		" auto or enabled should be treated as disabled"
	_ = complete(context.Background(), &apiv1.Config{
		AcornDNS: &s,
	}, nil)
	// if a lookup is going to happen this method would panic as the getter is nil
}

func TestAcornDNSStates(t *testing.T) {
	s := "disabled"
	conf := &apiv1.Config{
		AcornDNS: &s,
	}
	err := complete(context.Background(), conf, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Empty(t, conf.ClusterDomains)

	tests := []struct {
		name                   string
		expectedClusterDomains []string
		conf                   *apiv1.Config
		prepare                func(f *mocks.MockReader)
	}{
		{
			// acornDNS is explicitly disabled, expect no clusterDomain to be returned
			name: "acornDNS disabled expect no clusterdomains",
			conf: &apiv1.Config{
				AcornDNS: &[]string{"disabled"}[0], // hack for one-liner string pointer
			},
			expectedClusterDomains: nil,
		},
		{
			// acornDNS is explicitly disabled. User defined domain, expect just user defined domain
			name: "acornDNS disabled expect custom clusterdomains",
			conf: &apiv1.Config{
				AcornDNS:       &[]string{"disabled"}[0],
				ClusterDomains: []string{".custom.com"},
			},
			expectedClusterDomains: []string{".custom.com"},
		},
		{
			// acornDNS is in "auto" mode. No user configured domain, expect local as a fallback
			name: "acornDNS auto expect local clusterdomain",
			conf: &apiv1.Config{
				AcornDNS: &[]string{"auto"}[0],
			},
			expectedClusterDomains: []string{".local.on-acorn.io"},
			prepare: func(f *mocks.MockReader) {
				f.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				f.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			// acornDNS is in "auto" mode, but user configured a domain, expect just the user configured domain
			name: "acornDNS auto expect custom clusterdomain",
			conf: &apiv1.Config{
				AcornDNS:       &[]string{"auto"}[0],
				ClusterDomains: []string{".custom.com"},
			},
			expectedClusterDomains: []string{".custom.com"},
			prepare: func(f *mocks.MockReader) {
				f.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				f.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			r := mocks.NewMockReader(ctrl)
			if tt.prepare != nil {
				tt.prepare(r)
			}

			err := complete(context.Background(), tt.conf, r)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.expectedClusterDomains, tt.conf.ClusterDomains)
		})
	}
}

func TestMergeConfigWithZeroLengthStringsArrayShouldNotOverride(t *testing.T) {
	oldConfig := &apiv1.Config{
		AllowUserAnnotations:        []string{"foo"},
		AllowUserLabels:             []string{"foo"},
		PropagateProjectLabels:      []string{"foo"},
		PropagateProjectAnnotations: []string{"foo"},
		ClusterDomains:              []string{".acorn.io"},
	}

	newConfig := &apiv1.Config{
		AllowUserAnnotations:        []string{},
		AllowUserLabels:             []string{},
		PropagateProjectLabels:      []string{},
		PropagateProjectAnnotations: []string{},
		ClusterDomains:              []string{},
	}

	result := merge(oldConfig, newConfig)
	assert.Equal(t, []string{"foo"}, result.AllowUserAnnotations)
	assert.Equal(t, []string{"foo"}, result.AllowUserLabels)
	assert.Equal(t, []string{"foo"}, result.PropagateProjectAnnotations)
	assert.Equal(t, []string{"foo"}, result.PropagateProjectLabels)
	assert.Equal(t, []string{".acorn.io"}, result.ClusterDomains)
}

func TestMergeConfigWithActualValueStringsArrayShouldOverride(t *testing.T) {
	oldConfig := &apiv1.Config{
		AllowUserAnnotations:        []string{"foo"},
		AllowUserLabels:             []string{"foo"},
		PropagateProjectLabels:      []string{"foo"},
		PropagateProjectAnnotations: []string{"foo"},
		ClusterDomains:              []string{".acorn.io"},
	}

	newConfig := &apiv1.Config{
		AllowUserAnnotations:        []string{"bar"},
		AllowUserLabels:             []string{"bar"},
		PropagateProjectLabels:      []string{"bar"},
		PropagateProjectAnnotations: []string{"bar", "brah"},
		ClusterDomains:              []string{"bar.acorn.io"},
	}

	result := merge(oldConfig, newConfig)
	assert.Equal(t, []string{"bar"}, result.AllowUserAnnotations)
	assert.Equal(t, []string{"bar"}, result.AllowUserLabels)
	assert.Equal(t, []string{"bar"}, result.PropagateProjectLabels)
	assert.Equal(t, []string{"bar", "brah"}, result.PropagateProjectAnnotations)
	assert.Equal(t, []string{".bar.acorn.io"}, result.ClusterDomains)
}

func TestMergeConfigWithNilStringsArrayShouldNotOverride(t *testing.T) {
	oldConfig := &apiv1.Config{
		AllowUserAnnotations:        []string{"foo"},
		AllowUserLabels:             []string{"foo"},
		PropagateProjectLabels:      []string{"foo"},
		PropagateProjectAnnotations: []string{"foo"},
		ClusterDomains:              []string{".acorn.io"},
	}

	newConfig := &apiv1.Config{
		AllowUserAnnotations:        nil,
		AllowUserLabels:             nil,
		PropagateProjectLabels:      nil,
		PropagateProjectAnnotations: nil,
		ClusterDomains:              nil,
	}

	result := merge(oldConfig, newConfig)
	assert.Equal(t, []string{"foo"}, result.AllowUserAnnotations)
	assert.Equal(t, []string{"foo"}, result.AllowUserLabels)
	assert.Equal(t, []string{"foo"}, result.PropagateProjectLabels)
	assert.Equal(t, []string{"foo"}, result.PropagateProjectAnnotations)
	assert.Equal(t, []string{".acorn.io"}, result.ClusterDomains)
}

func TestMergeConfigWithEmptyStringStringsArrayShouldOverrideToNil(t *testing.T) {
	oldConfig := &apiv1.Config{
		AllowUserAnnotations:        []string{"foo"},
		AllowUserLabels:             []string{"foo"},
		PropagateProjectLabels:      []string{"foo"},
		PropagateProjectAnnotations: []string{"foo"},
		ClusterDomains:              []string{".acorn.io"},
	}

	newConfig := &apiv1.Config{
		AllowUserAnnotations:        []string{""},
		AllowUserLabels:             []string{""},
		PropagateProjectLabels:      []string{""},
		PropagateProjectAnnotations: []string{""},
		ClusterDomains:              []string{""},
	}

	result := merge(oldConfig, newConfig)
	assert.Nil(t, result.AllowUserAnnotations)
	assert.Nil(t, result.AllowUserLabels)
	assert.Nil(t, result.PropagateProjectLabels)
	assert.Nil(t, result.PropagateProjectAnnotations)
	assert.Nil(t, result.ClusterDomains)
}
