package buildserver

import (
	"context"
	"net/http"
	"strings"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build"
	"github.com/acorn-io/acorn/pkg/buildclient"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/acorn/pkg/imagesystem"
	"github.com/acorn-io/acorn/pkg/k8schannel"
	"github.com/acorn-io/acorn/pkg/pullsecret"
	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/watcher"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Server struct {
	uuid            string
	namespace       string
	client          kclient.WithWatch
	pubKey, privKey *[32]byte
}

type Token struct {
	BuilderUUID string                     `json:"builderUUID,omitempty"`
	Time        metav1.Time                `json:"time,omitempty"`
	Build       v1.AcornImageBuildInstance `json:"build,omitempty"`
	PushRepo    string                     `json:"pushRepo,omitempty"`
}

func NewServer(uuid, namespace string, pubKey, privKey [32]byte, client kclient.WithWatch) *Server {
	return &Server{
		uuid:      uuid,
		namespace: namespace,
		pubKey:    &pubKey,
		privKey:   &privKey,
		client:    client,
	}
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	err := s.serveHTTP(rw, req)
	if err != nil {
		logrus.Errorf("Build failed: %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte(err.Error()))
	}
}

func (s *Server) serveHTTP(rw http.ResponseWriter, req *http.Request) error {
	if strings.HasPrefix(req.URL.Path, "/ping") {
		_, err := rw.Write([]byte("pong"))
		return err
	}

	token, err := GetToken(req, s.uuid, s.pubKey, s.privKey)
	if err != nil {
		logrus.Errorf("Invalid token: %v", err)
		rw.WriteHeader(http.StatusUnauthorized)
		_, _ = rw.Write([]byte(err.Error()))
		return nil
	}

	if s.namespace != "" {
		token.Build.Namespace = s.namespace
	}

	conn, err := k8schannel.Upgrader.Upgrade(rw, req, nil)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte(err.Error()))
		return nil
	}
	defer conn.Close()

	k8schannel.AddCloseHandler(conn)

	m := buildclient.NewWebsocketMessages(conn)
	m.Start(req.Context())

	logrus.Infof("Starting build [%s/%s] [%s]", token.Build.Namespace, token.Build.Name, token.Build.UID)
	image, err := s.build(req.Context(), m, token)
	if err == nil {
		_ = m.Send(&buildclient.Message{
			AppImage: image,
		})
		logrus.Infof("Build succeeded [%s/%s] [%s]: %v", token.Build.Namespace, token.Build.Name, token.Build.UID, image.ID)
	} else {
		_ = m.Send(&buildclient.Message{
			Error: err.Error(),
		})
		logrus.Errorf("Build failed [%s/%s] [%s]: %v", token.Build.Namespace, token.Build.Name, token.Build.UID, err)
	}

	return nil
}

func retryOnConflict(f func() error) error {
	var err error
	for i := 0; i < 5; i++ {
		err = f()
		if apierrors.IsConflict(err) {
			logrus.Infof("Conflict retrying: %v", err)
			time.Sleep(time.Second)
			continue
		}
		return err
	}
	return err
}

func (s *Server) build(ctx context.Context, messages buildclient.Messages, token *Token) (*v1.AppImage, error) {
	if err := retryOnConflict(func() error {
		return s.recordBuildStart(ctx, &token.Build)
	}); err != nil {
		return nil, err
	}
	keychain, err := pullsecret.Keychain(ctx, s.client, token.Build.Namespace)
	if err != nil {
		return nil, err
	}
	image, err := build.Build(ctx, messages, token.PushRepo, token.Build.Spec, keychain)
	if err != nil {
		_ = s.recordBuildError(ctx, &token.Build, err)
		return nil, err
	}

	if err := retryOnConflict(func() error {
		return s.recordBuild(ctx, token.PushRepo, &token.Build, image)
	}); err != nil {
		return nil, err
	}
	return image, nil
}

func (s *Server) recordBuildStart(ctx context.Context, build *v1.AcornImageBuildInstance) error {
	recordedBuild := &v1.AcornImageBuildInstance{}
	err := s.client.Get(ctx, kclient.ObjectKeyFromObject(build), recordedBuild)
	if apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	condition.Setter(recordedBuild, nil, v1.AcornImageBuildInstanceConditionBuild).Unknown("Building")
	recordedBuild.Status.ObservedGeneration = build.Generation
	return s.client.Status().Update(ctx, recordedBuild)
}

func (s *Server) recordBuildError(ctx context.Context, build *v1.AcornImageBuildInstance, buildError error) error {
	recordedBuild := &v1.AcornImageBuildInstance{}
	err := s.client.Get(ctx, kclient.ObjectKeyFromObject(build), recordedBuild)
	if apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	recordedBuild.Status.BuildError = buildError.Error()
	condition.Setter(recordedBuild, nil, v1.AcornImageBuildInstanceConditionBuild).Error(buildError)
	recordedBuild.Status.ObservedGeneration = build.Generation
	return s.client.Status().Update(ctx, recordedBuild)
}

func (s *Server) recordBuild(ctx context.Context, recordRepo string, build *v1.AcornImageBuildInstance, image *v1.AppImage) error {
	if imagesystem.IsClusterInternalRegistryAddressReference(recordRepo) {
		recordRepo = ""
	}

	imageInst := new(v1.ImageInstance)
	if err := s.client.Get(ctx, kclient.ObjectKey{Name: image.ID, Namespace: build.Namespace}, imageInst); err == nil && imageInst.Remote {
		// Ensure that the image is not remote since  it was built locally.
		// This can happen if an image was pulled from a remote registry and then built locally.
		imageInst.Remote = false
		if err = s.client.Update(ctx, imageInst); err != nil {
			return err
		}
	} else if apierrors.IsNotFound(err) {
		if err = apply.New(s.client).Ensure(ctx, &v1.ImageInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      image.ID,
				Namespace: build.Namespace,
			},
			Repo:   recordRepo,
			Digest: image.Digest,
		}); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	recordedBuild := &v1.AcornImageBuildInstance{}
	err := s.client.Get(ctx, kclient.ObjectKeyFromObject(build), recordedBuild)
	if apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	condition.Setter(recordedBuild, nil, v1.AcornImageBuildInstanceConditionBuild).Success()
	recordedBuild.Status.AppImage = *image
	recordedBuild.Status.ObservedGeneration = build.Generation
	if err := s.client.Status().Update(ctx, recordedBuild); err != nil {
		return err
	}
	logrus.Infof("Waiting for build %s/%s to be recorded", recordedBuild.Name, recordedBuild.Namespace)
	_, err = watcher.New[*v1.AcornImageBuildInstance](s.client).ByObject(ctx, recordedBuild, func(obj *v1.AcornImageBuildInstance) (bool, error) {
		return obj.Status.Recorded, nil
	})
	return err
}
