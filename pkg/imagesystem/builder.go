package imagesystem

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/base64"
	"fmt"

	name2 "github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/digest"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/google/go-containerregistry/pkg/name"
	"golang.org/x/crypto/nacl/box"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetBuildPushRepoForNamespace(ctx context.Context, c client.Reader, namespace string) (name.Repository, error) {
	cfg, err := config.Get(ctx, c)
	if err != nil {
		return name.Repository{}, err
	}
	if *cfg.InternalRegistryPrefix != "" {
		return name.NewRepository(*cfg.InternalRegistryPrefix + namespace)
	}

	return name.NewRepository(fmt.Sprintf("127.0.0.1:%d/acorn/%s", system.RegistryPort, namespace))
}

func GetBuilderDeploymentName(ctx context.Context, c client.Reader, builderName, builderNamespace string) (string, error) {
	cfg, err := config.Get(ctx, c)
	if err != nil {
		return "", err
	}
	name := system.BuildKitName
	if *cfg.BuilderPerProject {
		name = name2.SafeConcatName("bld", builderName, builderNamespace, digest.SHA256(builderName, builderNamespace)[:8])
	}
	return name, nil
}

func GetBuilderKeys(ctx context.Context, c client.Reader, namespace, name string) (string, string, error) {
	var (
		secret          = &corev1.Secret{}
		pubKey, privKey string
	)

	err := c.Get(ctx, router.Key(namespace, name), secret)
	if apierrors.IsNotFound(err) {
		pubData, privData, err := box.GenerateKey(cryptorand.Reader)
		if err != nil {
			return "", "", err
		}
		pubKey = base64.RawURLEncoding.EncodeToString(pubData[:])
		privKey = base64.RawURLEncoding.EncodeToString(privData[:])
	} else if err != nil {
		return "", "", err
	} else {
		pubKey = string(secret.Data["pub"])
		privKey = string(secret.Data["priv"])
	}

	return pubKey, privKey, nil
}
