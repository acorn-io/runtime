package publish

import (
	"errors"
	"reflect"
	"testing"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestToEndpoint(t *testing.T) {
	type args struct {
		domain      string
		serviceName string
		pattern     string
		appInstance *v1.AppInstance
	}
	tests := []struct {
		name string
		args args

		wantEndpoint string
		wantErr      error
	}{
		{
			name: "\"on-acorn.io no -\" pattern set",
			args: args{
				domain:      "domain.on-acorn.io",
				serviceName: "app-test",
				pattern:     "{{hashConcat 8 .Container .App .Namespace | truncate}}.{{.ClusterDomain}}",
				appInstance: &v1.AppInstance{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: "green-star"},
					Spec:       v1.AppInstanceSpec{},
					Status:     v1.AppInstanceStatus{},
				},
			},
			wantEndpoint: "app-test-green-star-b19d0b34.domain.on-acorn.io",
		},
		{
			name: "\"on-acorn.io no -\" pattern set with 12 characters",
			args: args{
				domain:      "domain.on-acorn.io",
				serviceName: "app-test",
				pattern:     "{{hashConcat 12 .Container .App .Namespace | truncate}}.{{.ClusterDomain}}",
				appInstance: &v1.AppInstance{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: "green-star"},
					Spec:       v1.AppInstanceSpec{},
					Status:     v1.AppInstanceStatus{},
				},
			},
			wantEndpoint: "app-test-green-star-b19d0b346674.domain.on-acorn.io",
		},
		{
			name: "\"on-acorn.io no -\" pattern set with less than two parameters should return empty string",
			args: args{
				domain:      "domain.on-acorn.io",
				serviceName: "app-test",
				pattern:     "{{hashConcat 8 .Container | truncate}}.{{.ClusterDomain}}",
				appInstance: &v1.AppInstance{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: "green-star"},
					Spec:       v1.AppInstanceSpec{},
					Status:     v1.AppInstanceStatus{},
				},
			},
			wantEndpoint: ".domain.on-acorn.io",
		},
		{
			name: "\"custom domain\" pattern set",
			args: args{
				domain:      "domain.custom-domain.io",
				serviceName: "app-test",
				pattern:     "{{.Container}}.{{.App}}.{{.Namespace}}.{{.ClusterDomain}}",
				appInstance: &v1.AppInstance{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: "green-star", Namespace: "namespace"},
					Spec:       v1.AppInstanceSpec{},
					Status:     v1.AppInstanceStatus{},
				},
			},
			wantEndpoint: "app-test.green-star.namespace.domain.custom-domain.io",
		},
		{
			name: "\"custom domain default service\" pattern set",
			args: args{
				domain:      "domain.custom-domain.io",
				serviceName: "default",
				pattern:     "{{.App}}.{{.Namespace}}.{{.ClusterDomain}}",
				appInstance: &v1.AppInstance{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: "green-star", Namespace: "namespace"},
					Spec:       v1.AppInstanceSpec{},
					Status:     v1.AppInstanceStatus{},
				},
			},
			wantEndpoint: "green-star.namespace.domain.custom-domain.io",
		},
		{
			name: "friendly pattern set",
			args: args{
				domain:      "custom-domain.io",
				serviceName: "app-test",
				pattern:     "{{.Container}}.{{.App}}.{{.Namespace}}.{{.ClusterDomain}}",
				appInstance: &v1.AppInstance{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: "green-star", Namespace: "namespace"},
					Spec:       v1.AppInstanceSpec{},
					Status:     v1.AppInstanceStatus{},
				},
			},
			wantEndpoint: "app-test.green-star.namespace.custom-domain.io",
		},
		{
			name: "lets encrypt pattern set",
			args: args{
				domain:      "custom-domain.io",
				serviceName: "app-test",
				pattern:     "{{hashConcat 8 .Container .App .Namespace | truncate}}.{{.ClusterDomain}}",
				appInstance: &v1.AppInstance{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: "green-star", Namespace: "namespace"},
					Spec:       v1.AppInstanceSpec{},
					Status:     v1.AppInstanceStatus{},
				},
			},
			wantEndpoint: "app-test-green-star-49eba2c9.custom-domain.io",
		},
		{
			name: "custom pattern set",
			args: args{
				domain:      "custom-domain.io",
				serviceName: "app-test",
				pattern:     "{{.Container}}-{{.App}}.{{.Namespace}}-cluster.{{.ClusterDomain}}",
				appInstance: &v1.AppInstance{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: "green-star", Namespace: "namespace"},
					Spec:       v1.AppInstanceSpec{},
					Status:     v1.AppInstanceStatus{},
				},
			},
			wantEndpoint: "app-test-green-star.namespace-cluster.custom-domain.io",
		},
		{
			name: "no pattern set",
			args: args{
				domain:      "custom-domain.io",
				serviceName: "app-test",
				pattern:     "",
				appInstance: &v1.AppInstance{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: "green-star", Namespace: "namespace"},
					Spec:       v1.AppInstanceSpec{},
					Status:     v1.AppInstanceStatus{},
				},
			},
			wantEndpoint: "app-test-green-star-49eba2c9.custom-domain.io",
		},
		{
			name: "bad pattern set",
			args: args{
				domain:      "custom-domain.io",
				serviceName: "app-test",
				pattern:     "{{.Foo}}-{{.Bar}}.{{.Baz}}-cluster.{{.Bat}}",
				appInstance: &v1.AppInstance{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: "green-star", Namespace: "namespace"},
					Spec:       v1.AppInstanceSpec{},
					Status:     v1.AppInstanceStatus{},
				},
			},
			wantEndpoint: "",
			wantErr:      ErrInvalidPattern,
		},
		{
			name: "parsed pattern's segment exceeds maximum length",
			args: args{
				domain:      "custom-domain.io",
				serviceName: "app-name-that-is-very-long-and-should-cause-issues",
				pattern:     "{{hashConcat 8 .Container .App .Namespace | truncate}}-{{.Namespace}}.{{.ClusterDomain}}",
				appInstance: &v1.AppInstance{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: "green-star", Namespace: "namespace"},
					Spec:       v1.AppInstanceSpec{},
					Status:     v1.AppInstanceStatus{},
				},
			},
			wantEndpoint: "",
			wantErr:      ErrSegmentExceededMaxLength,
		},
		{
			name: "parsed pattern's segment exceeds maximum length and should be truncated",
			args: args{
				domain:      "custom-domain.io",
				serviceName: "app-name-that-is-very-long-and-should-cause-issues",
				pattern:     config.DefaultHttpEndpointPattern,
				appInstance: &v1.AppInstance{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: "green-star", Namespace: "namespace"},
					Spec:       v1.AppInstanceSpec{},
					Status:     v1.AppInstanceStatus{},
				},
			},
			wantEndpoint: "app-name-that-is-very-long-and-should-cause-issues-green-59d6e2.custom-domain.io",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEndpoint, err := toHTTPEndpointHostname(tt.args.pattern, tt.args.domain, tt.args.serviceName, tt.args.appInstance.GetName(), tt.args.appInstance.GetNamespace())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("toHTTPEndpointHostname() error = %v, want %v", err, tt.wantErr)
			}

			if gotEndpoint != tt.wantEndpoint {
				t.Errorf("toHTTPEndpointHostname() = %v, want %v", gotEndpoint, tt.wantEndpoint)
			}
		})
	}
}

func TestValidateEndpointPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr error
	}{
		{
			name:    "valid",
			pattern: "{{hashConcat 8 .Container .App .Namespace | truncate}}.{{.ClusterDomain}}",
			wantErr: nil,
		},
		{
			name:    "invalid constructed domain",
			pattern: "{{hashConcat 8 .Container .App .Namespace | truncate}}.$INVALID$.{{.ClusterDomain}}",
			wantErr: ErrInvalidPattern,
		},
		{
			name:    "invalid parameters",
			pattern: "{{hashConcat .Container .App .Namespace | truncate}}.$INVALID$.{{.ClusterDomain}}",
			wantErr: ErrInvalidPattern,
		},
		{
			name:    "referencing invalid variables",
			pattern: "{{hashConcat 8 .Foo .Bar .Baz | truncate}}-{{.Namespace}}.{{.ClusterDomain}}",
			wantErr: ErrInvalidPattern,
		},
		{
			name:    "invalid go template",
			pattern: "{{.InvalidReference}}-{{.App}}-{{.Hash}}.{{.ClusterDomain}}",
			wantErr: ErrInvalidPattern,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEndpointPattern(tt.pattern)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("toHTTPEndpointHostname() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func Test_setupCertManager(t *testing.T) {
	type args struct {
		serviceName string
		annotations map[string]string
		rules       []networkingv1.IngressRule
		tls         []networkingv1.IngressTLS
	}
	tests := []struct {
		name string
		args args
		want []networkingv1.IngressTLS
	}{
		{
			name: "no annotation",
			args: args{
				serviceName: "foo",
				annotations: map[string]string{},
				tls:         []networkingv1.IngressTLS{{Hosts: []string{"host"}}},
			},
			want: []networkingv1.IngressTLS{{Hosts: []string{"host"}}},
		},
		{
			name: "annotation and tls found, prefer annotation",
			args: args{
				serviceName: "foo",
				annotations: map[string]string{
					"cert-manager.io/cluster-issuer": "foo",
				},
				rules: []networkingv1.IngressRule{{Host: "host1"}},
				tls:   []networkingv1.IngressTLS{{Hosts: []string{"host"}}},
			},
			want: []networkingv1.IngressTLS{{Hosts: []string{"host1"}, SecretName: "foo-cm-cert-1"}},
		},
		{
			name: "cluster-issuer annotation found",
			args: args{
				serviceName: "foo",
				annotations: map[string]string{
					"cert-manager.io/cluster-issuer": "foo",
				},
				rules: []networkingv1.IngressRule{{Host: "host1"}},
			},
			want: []networkingv1.IngressTLS{{
				Hosts:      []string{"host1"},
				SecretName: "foo-cm-cert-1",
			}},
		},
		{
			name: "issuer annotation found",
			args: args{
				serviceName: "foo",
				annotations: map[string]string{
					"cert-manager.io/issuer": "foo",
				},
				rules: []networkingv1.IngressRule{{Host: "host1"}},
			},
			want: []networkingv1.IngressTLS{{
				Hosts:      []string{"host1"},
				SecretName: "foo-cm-cert-1",
			}},
		},
		{
			name: "two hosts",
			args: args{
				serviceName: "foo",
				annotations: map[string]string{
					"cert-manager.io/issuer": "foo",
				},
				rules: []networkingv1.IngressRule{{Host: "host1"}, {Host: "host2"}},
			},
			want: []networkingv1.IngressTLS{{
				Hosts:      []string{"host1"},
				SecretName: "foo-cm-cert-1",
			}, {
				Hosts:      []string{"host2"},
				SecretName: "foo-cm-cert-2",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := setupCertManager(tt.args.serviceName, tt.args.annotations, tt.args.rules, tt.args.tls); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("setupCertManager() = %v, want %v", got, tt.want)
			}
		})
	}
}
