package log

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	appWithLinkerdProxy = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-with-linkerd-pod",
			Namespace: "app-namespace",
			Labels: map[string]string{
				labels.AcornAppName:       "app-name",
				labels.AcornAppNamespace:  "acorn",
				labels.AcornContainerName: "nginx",
				labels.AcornManaged:       "true",
			},
			Annotations: map[string]string{
				labels.AcornContainerSpec: "{\"build\":{\"baseImage\":\"nginx:latest\",\"context\":\".\",\"dockerfile\":\"Dockerfile\"},\"image\":\"sha256:d9b6a5b0b1711fd71903e7785ada61fde440b880e263053d95c9028f761f5e17\",\"permissions\":{},\"ports\":[{\"port\":80,\"protocol\":\"http\",\"targetPort\":80}],\"probes\":null,\"sidecars\":{\"sidecar\":{\"build\":{\"baseImage\":\"ubuntu:latest\",\"context\":\".\",\"dockerfile\":\"Dockerfile\"},\"command\":[\"bash\",\"-c\",\"echo hello world \\u0026\\u0026 sleep 999\"],\"image\":\"sha256:b630a928dfec0286ae493bb0e79cae3ee77d811064c8fb0734cb5badc1367338\",\"permissions\":{},\"probes\":null}}}",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "linkerd-proxy"},
				{Name: "nginx"},
				{Name: "sidecar"},
			},
		},
	}
	jobWithLinkerdProxy = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-with-linkerd-pod",
			Namespace: "app-namespace",
			Labels: map[string]string{
				labels.AcornAppName:      "app-with-job",
				labels.AcornAppNamespace: "acorn",
				labels.AcornJobName:      "busybox",
				labels.AcornManaged:      "true",
			},
			Annotations: map[string]string{
				labels.AcornContainerSpec: "{\"build\":{\"baseImage\":\"busybox:latest\",\"context\":\".\",\"dockerfile\":\"Dockerfile\"},\"command\":[\"sh\",\"-c\",\"echo hello world\"],\"image\":\"sha256:0b19e271137ea1aec3cee91eff0b0cc1924bbd2a507b0a213a3028ed55c20d21\",\"permissions\":{},\"probes\":null}",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "linkerd-proxy"},
				{Name: "busybox"},
			},
		},
	}
)

func TestMatchesContainer(t *testing.T) {
	type args struct {
		pod       *corev1.Pod
		container corev1.Container
		options   *Options
	}

	tests := []struct {
		name           string
		args           args
		expectedResult bool
	}{
		{
			name: "nginx-match",
			args: args{
				pod:       appWithLinkerdProxy,
				container: corev1.Container{Name: "nginx"},
				options:   nil,
			},
			expectedResult: true,
		},
		{
			name: "sidecar-match",
			args: args{
				pod:       appWithLinkerdProxy,
				container: corev1.Container{Name: "sidecar"},
				options:   nil,
			},
			expectedResult: true,
		},
		{
			name: "linkerd-proxy-no-match",
			args: args{
				pod:       appWithLinkerdProxy,
				container: corev1.Container{Name: "linkerd-proxy"},
				options:   nil,
			},
			expectedResult: false,
		},
		{
			name: "app-name.app-with-linkerd-pod-match",
			args: args{
				pod:       appWithLinkerdProxy,
				container: corev1.Container{Name: "nginx"},
				options: &Options{
					ContainerReplica: "app-name.app-with-linkerd-pod",
				},
			},
			expectedResult: true,
		},
		{
			name: "app-name.app-with-linkerd-pod.nginx-match",
			args: args{
				pod:       appWithLinkerdProxy,
				container: corev1.Container{Name: "nginx"},
				options: &Options{
					ContainerReplica: "app-name.app-with-linkerd-pod.nginx",
				},
			},
			expectedResult: true,
		},
		{
			name: "app-name.app-with-linkerd-pod.sidecar-match",
			args: args{
				pod:       appWithLinkerdProxy,
				container: corev1.Container{Name: "sidecar"},
				options: &Options{
					ContainerReplica: "app-name.app-with-linkerd-pod.sidecar",
				},
			},
			expectedResult: true,
		},
		{
			name: "app-name.nonexistent-no-match",
			args: args{
				pod:       appWithLinkerdProxy,
				container: corev1.Container{Name: "nonexistent"},
				options: &Options{
					ContainerReplica: "app-name.nonexistent",
				},
			},
			expectedResult: false,
		},
		{
			name: "busybox-match",
			args: args{
				pod:       jobWithLinkerdProxy,
				container: corev1.Container{Name: "busybox"},
				options:   nil,
			},
			expectedResult: true,
		},
		{
			name: "linkerd-proxy-job-no-match",
			args: args{
				pod:       jobWithLinkerdProxy,
				container: corev1.Container{Name: "linkerd-proxy"},
				options:   nil,
			},
			expectedResult: false,
		},
		{
			name: "app-with-job.job-with-linkerd-pod-match",
			args: args{
				pod:       jobWithLinkerdProxy,
				container: corev1.Container{Name: "busybox"},
				options: &Options{
					ContainerReplica: "app-with-job.job-with-linkerd-pod",
				},
			},
			expectedResult: true,
		},
		{
			name: "app-with-job.job-with-linkerd-pod.busybox-match",
			args: args{
				pod:       jobWithLinkerdProxy,
				container: corev1.Container{Name: "busybox"},
				options: &Options{
					ContainerReplica: "app-with-job.job-with-linkerd-pod.busybox",
				},
			},
			expectedResult: true,
		},
		{
			name: "app-with-job.nonexistent-no-match",
			args: args{
				pod:       jobWithLinkerdProxy,
				container: corev1.Container{Name: "nonexistent"},
				options: &Options{
					ContainerReplica: "app-with-job.nonexistent",
				},
			},
			expectedResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := matchesContainer(test.args.pod, test.args.container, test.args.options)
			assert.EqualValues(t, test.expectedResult, result)
		})
	}
}
