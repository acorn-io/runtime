package local

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/acorn-io/baaah/pkg/yaml"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/install"
	"github.com/acorn-io/runtime/pkg/local/webhook"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/acorn-io/z"
	"github.com/google/go-containerregistry/pkg/name"
	ggcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func ServerRun(ctx context.Context) error {
	if os.Getuid() != 0 {
		return fmt.Errorf("must run as root")
	}

	if _, err := os.Stat("/.dockerenv"); err != nil {
		return fmt.Errorf("must be ran in a docker container: %w", err)
	}

	f, err := os.Open("/dev/kmsg")
	if err != nil {
		return fmt.Errorf("must be ran in a privileged docker container: %w", err)
	}

	_ = f.Close()

	if err := os.WriteFile("/etc/machine-id", []byte(system.LocalImage), 0644); err != nil {
		return err
	}

	c, err := NewContainer()
	if err != nil {
		return err
	}

	if err = c.DeletePorts(ctx); err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	if err = install.PrintObjects("acorn-local", &install.Options{
		Output:                   buf,
		IncludeLocalEnvResources: true,
		Config: apiv1.Config{
			IngressClassName:             z.Pointer("traefik"),
			SetPodSecurityEnforceProfile: z.Pointer(false),
			IgnoreResourceRequirements:   z.Pointer(true),
		},
	}); err != nil {
		return err
	}

	objs, err := yaml.ToObjects(buf)
	if err != nil {
		return err
	}

	for i, obj := range objs {
		u := obj.(*unstructured.Unstructured)
		if u.GetKind() == "Deployment" {
			data, err := json.Marshal(u)
			if err != nil {
				return err
			}
			var dep appsv1.Deployment
			if err := json.Unmarshal(data, &dep); err != nil {
				return err
			}
			webhook.PatchPodSpec(&dep.Spec.Template.Spec)
			objs[i] = &dep
		}
		if u.GetKind() == "ConfigMap" && u.GetName() == "coredns" {
			cmd := exec.Command("/bin/sh", "-c", "ip addr show dev eth0 | grep inet | cut -f1 -d/ | awk '{print $2}'")
			out, err := cmd.CombinedOutput()
			if err != nil {
				return err
			}
			if err := unstructured.SetNestedField(u.Object, fmt.Sprintf("%s acorn-node\n", out), "data", "NodeHosts"); err != nil {
				return err
			}
		}
	}

	data, err := yaml.Export(scheme.Scheme, objs...)
	if err != nil {
		return err
	}

	if err = os.MkdirAll("/var/lib/rancher/k3s/server/manifests", 0755); err != nil {
		return err
	}

	if err = os.WriteFile("/var/lib/rancher/k3s/server/manifests/acorn.yaml", data, 0655); err != nil {
		return err
	}

	if _, err = os.Stat("/sys/fs/cgroup/cgroup.controllers"); err == nil {
		cmd := exec.Command("/bin/sh", "-c", `
mkdir -p /sys/fs/cgroup/init
busybox xargs -rn1 < /sys/fs/cgroup/cgroup.procs > /sys/fs/cgroup/init/cgroup.procs || :
sed -e 's/ / +/g' -e 's/^/+/' <"/sys/fs/cgroup/cgroup.controllers" >"/sys/fs/cgroup/cgroup.subtree_control"
`)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err = cmd.Run(); err != nil {
			return fmt.Errorf("failed to setup cgroups: %w", err)
		}
	}

	ref, err := name.NewTag(system.LocalImageBind)
	if err != nil {
		return err
	}

	img, err := buildImage()
	if err != nil {
		return err
	}

	if err = os.MkdirAll("/var/lib/rancher/k3s/agent/images", 0755); err != nil {
		return err
	}

	if err := tarball.WriteToFile("/var/lib/rancher/k3s/agent/images/empty.tar", ref, img); err != nil {
		return err
	}

	return syscall.Exec("/bin/k3s", []string{"k3s", "server"}, os.Environ())
}

func buildImage() (ggcrv1.Image, error) {
	layer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		out := &bytes.Buffer{}
		t := tar.NewWriter(out)
		for _, dir := range []string{"wd", "tmp", "var", "var/lib", "etc", "etc/nginx", "var/log", "var/log/nginx",
			"run", "var/cache", "var/cache/nginx"} {
			err := t.WriteHeader(&tar.Header{
				Typeflag: tar.TypeDir,
				Name:     dir,
				Mode:     0777,
			})
			if err != nil {
				return nil, err
			}
		}
		if err := t.Close(); err != nil {
			return nil, err
		}
		return io.NopCloser(out), nil
	})
	if err != nil {
		return nil, err
	}

	img, err := mutate.Config(empty.Image, ggcrv1.Config{
		Entrypoint: []string{
			"/usr/local/bin/acorn",
		},
		Env: []string{
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		Volumes: map[string]struct{}{
			"/var/lib/rancher/k3s": {},
			"/var/lib/buildkit":    {},
		},
		WorkingDir: "/wd",
		StopSignal: "SIGTERM",
	})
	if err != nil {
		return nil, err
	}

	return mutate.AppendLayers(img, layer)
}
