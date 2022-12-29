package buildserver

import (
	"context"
	"net/http"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build"
	"github.com/acorn-io/acorn/pkg/buildclient"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/acorn/pkg/images"
	"github.com/acorn-io/acorn/pkg/imagesystem"
	"github.com/acorn-io/acorn/pkg/k8schannel"
	"github.com/acorn-io/acorn/pkg/pullsecret"
	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Server struct {
	uuid            string
	namespace       string
	client          kclient.Client
	pubKey, privKey *[32]byte
}

type Token struct {
	BuilderUUID string                     `json:"builderUUID,omitempty"`
	Time        metav1.Time                `json:"time,omitempty"`
	Build       v1.AcornImageBuildInstance `json:"build,omitempty"`
	PushRepo    string                     `json:"pushRepo,omitempty"`
}

func NewServer(uuid, namespace string, pubKey, privKey [32]byte, client kclient.Client) *Server {
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

func (s *Server) build(ctx context.Context, messages buildclient.Messages, token *Token) (*v1.AppImage, error) {
	if err := s.recordBuildStart(ctx, &token.Build); err != nil {
		return nil, err

	}
	keychain, err := pullsecret.Keychain(ctx, s.client, token.Build.Namespace)
	if err != nil {
		return nil, err
	}
	opts, err := images.GetAuthenticationRemoteOptions(ctx, s.client, token.Build.Namespace)
	if err != nil {
		return nil, err
	}
	image, err := build.Build(ctx, messages, token.PushRepo, &token.Build.Spec, keychain, opts...)
	if err != nil {
		_ = s.recordBuildError(ctx, &token.Build, err)
		return nil, err
	}

	return image, s.recordBuild(ctx, token.PushRepo, &token.Build, image)
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
	err := apply.New(s.client).Ensure(ctx, &v1.ImageInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      image.ID,
			Namespace: build.Namespace,
		},
		Repo:   recordRepo,
		Digest: image.Digest,
	})
	if err != nil {
		return err
	}

	recordedBuild := &v1.AcornImageBuildInstance{}
	err = s.client.Get(ctx, kclient.ObjectKeyFromObject(build), recordedBuild)
	if apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	condition.Setter(recordedBuild, nil, v1.AcornImageBuildInstanceConditionBuild).Success()
	recordedBuild.Status.AppImage = *image
	recordedBuild.Status.ObservedGeneration = build.Generation
	return s.client.Status().Update(ctx, recordedBuild)
}
