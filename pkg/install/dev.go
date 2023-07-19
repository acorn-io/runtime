package install

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/acorn-io/baaah/pkg/apply"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/dockerconfig"
	"github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/acorn-io/z"
	"github.com/google/go-containerregistry/pkg/name"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func devImage(imageName string) (string, error) {
	idfile, err := os.CreateTemp("", "acorn-build-id")
	if err != nil {
		return "", err
	}
	if err := idfile.Close(); err != nil {
		return "", err
	}
	defer os.Remove(idfile.Name())

	cmd := exec.Command("docker", "build", "-t", imageName, "--iidfile", idfile.Name(), "--push", ".")
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	idData, err := os.ReadFile(idfile.Name())
	if err != nil {
		return "", err
	}

	cmd = exec.Command("docker", "image", "inspect", "--format={{ index .RepoDigests 0}}", string(idData))
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	d, err := cmd.Output()
	if err != nil {
		return "", err
	}

	digest, err := name.NewDigest(strings.TrimSpace(string(d)))
	if err != nil {
		return "", err
	}

	return digest.DigestStr(), nil
}

func toDevSecret(imageName string, cred *apiv1.RegistryAuth) (*corev1.Secret, error) {
	parsedImage, err := name.NewDigest(imageName)
	if err != nil {
		return nil, err
	}

	secret, err := dockerconfig.FromCredential(&apiv1.Credential{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dev-credential",
			Namespace: system.Namespace,
		},
		ServerAddress: parsedImage.RegistryStr(),
		Username:      cred.Username,
		Password:      z.Pointer(cred.Password),
	})
	if err != nil {
		return nil, err
	}

	if secret.Annotations == nil {
		secret.Annotations = map[string]string{}
	}
	secret.Annotations[labels.AcornCredential] = "true"

	return secret, nil
}

func Dev(ctx context.Context, imageName string, cred *apiv1.RegistryAuth, opts *Options) error {
	c, err := k8sclient.Default()
	if err != nil {
		return err
	}

	digest, err := devImage(imageName)
	if err != nil {
		return err
	}

	imageName += "@" + digest

	devSecret, err := toDevSecret(imageName, cred)
	if err != nil {
		return err
	}

	parsed, err := name.ParseReference(imageName)
	if err != nil {
		return err
	}

	opts.Config.InternalRegistryPrefix = z.Pointer(filepath.Dir(parsed.Context().RepositoryStr()) + "/")

	cm, err := config.AsConfigMap(&opts.Config)
	if err != nil {
		return err
	}
	cm.Name = system.DevConfigName

	if cm.Annotations == nil {
		cm.Annotations = map[string]string{}
	}

	cm.Annotations[labels.DevImageName] = imageName
	cm.Annotations[labels.DevCredentialName] = devSecret.Name
	cm.Annotations[labels.DevTTL] = time.Now().Add(8 * time.Hour).Format(time.RFC3339)

	if err := apply.New(c).Ensure(ctx, devSecret, cm); err != nil {
		return err
	}

	return Install(ctx, "", &Options{
		SkipChecks: true,
	})
}
