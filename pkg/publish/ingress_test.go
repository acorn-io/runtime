package publish

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_toPrefix(t *testing.T) {
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
			wantHostPrefix: "app-test-green-star-fefc5537672b",
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
			wantHostPrefix: "apptest-green-star-404b24abeb8c",
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
			wantHostPrefix: "app-test-greenstar-e735671e185a",
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
