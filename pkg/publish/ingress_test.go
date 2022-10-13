package publish

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestToPrefix(t *testing.T) {
	type args struct {
		domain      string
		serviceName string
		appInstance *v1.AppInstance
	}
	tests := []struct {
		name           string
		args           args
		wantHostPrefix string
	}{
		{
			name: "\"on-acorn.io\" Valid Args",
			args: args{
				domain:      "domain.on-acorn.io",
				serviceName: "app-test",
				appInstance: &v1.AppInstance{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: "green-star"},
					Spec:       v1.AppInstanceSpec{},
					Status:     v1.AppInstanceStatus{},
				},
			},
			wantHostPrefix: "app-test-green-star-6b72f64c5182",
		},
		{
			name: "\"on-acorn.io\" Service Name No -",
			args: args{
				domain:      "domain.on-acorn.io",
				serviceName: "apptest",
				appInstance: &v1.AppInstance{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: "green-star"},
					Spec:       v1.AppInstanceSpec{},
					Status:     v1.AppInstanceStatus{},
				},
			},
			wantHostPrefix: "apptest-green-star-75803e8503f9",
		},
		{
			name: "\"on-acorn.io\" AppInstance Name No -",
			args: args{
				domain:      "domain.on-acorn.io",
				serviceName: "app-test",
				appInstance: &v1.AppInstance{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: "greenstar"},
					Spec:       v1.AppInstanceSpec{},
					Status:     v1.AppInstanceStatus{},
				},
			},
			wantHostPrefix: "app-test-greenstar-c832271c0202",
		},
		{
			name: "\"custom domain\"",
			args: args{
				domain:      "domain.custom-domain.io",
				serviceName: "app-test",
				appInstance: &v1.AppInstance{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: "green-star", Namespace: "namespace"},
					Spec:       v1.AppInstanceSpec{},
					Status:     v1.AppInstanceStatus{},
				},
			},
			wantHostPrefix: "app-test.green-star.namespace",
		},
		{
			name: "\"custom domain\" default service name",
			args: args{
				domain:      "domain.custom-domain.io",
				serviceName: "default",
				appInstance: &v1.AppInstance{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: "green-star", Namespace: "namespace"},
					Spec:       v1.AppInstanceSpec{},
					Status:     v1.AppInstanceStatus{},
				},
			},
			wantHostPrefix: "green-star.namespace",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotHostPrefix := toPrefix(tt.args.domain, tt.args.serviceName, tt.args.appInstance); gotHostPrefix != tt.wantHostPrefix {
				t.Errorf("toPrefix() = %v, want %v", gotHostPrefix, tt.wantHostPrefix)
			}
		})
	}
}
