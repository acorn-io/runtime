package cosign

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"net/url"
	"testing"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

//go:embed testdata/validkey1.pub
var VALIDKEY1 string

//go:embed testdata/invalidkey1.pub
var INVALIDKEY1 string

// adapted from google/go-containerregistry (crane)
func loadOCIDir(path string) (partial.WithRawManifest, error) {
	l, err := layout.ImageIndexFromPath(path)
	if err != nil {
		return nil, fmt.Errorf("loading %s as OCI layout: %w", path, err)
	}

	m, err := l.IndexManifest()
	if err != nil {
		return nil, err
	}
	if len(m.Manifests) != 1 {
		return nil, fmt.Errorf("layout contains %d entries, consider --index", len(m.Manifests))
	}

	desc := m.Manifests[0]
	if desc.MediaType.IsImage() {
		return l.Image(desc.Digest)
	} else if desc.MediaType.IsIndex() {
		return l.ImageIndex(desc.Digest)
	}
	return nil, fmt.Errorf("layout contains non-image (mediaType: %q), consider --index", desc.MediaType)
}

// adapted from google/go-containerregistry (crane)
func pushOCIDir(path string, ref name.Reference) error {
	img, err := loadOCIDir(path)
	if err != nil {
		return err
	}

	var h regv1.Hash
	switch t := img.(type) {
	case regv1.Image:
		if err := remote.Write(ref, t); err != nil {
			return err
		}
		if h, err = t.Digest(); err != nil {
			return err
		}
	case regv1.ImageIndex:
		if err := remote.WriteIndex(ref, t); err != nil {
			return err
		}
		if h, err = t.Digest(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot push type (%T) to registry", img)
	}

	fmt.Println(h)

	return nil
}

// TODO: mock oci registry and signatures
func TestVerifySignature(t *testing.T) {
	imgPath := "./testdata/img.oci"
	imgDigestExpected := "sha256:245864d0312e7e33201eff111cfc071727f4eaa9edd10a395c367077e200cad2"

	sigTag := "sha256-245864d0312e7e33201eff111cfc071727f4eaa9edd10a395c367077e200cad2.sig"

	sigNotOKPath := "./testdata/sig_notok_1.oci"
	sigNotOKDigestExpected := "sha256:2660a9b71fe930cf09694b1cb802e8b8e7ab9f2d173d5d2663caf5f1481ae240"

	sigNotOK2Path := "./testdata/sig_notok_2.oci"
	sigNotOK2DigestExpected := "sha256:3c1e9781522d836a400b67ad65599e7ace6ebc4a739ee4eefc163127474d73b1"

	sigOK1Path := "./testdata/sig_ok_1.oci"
	sigOK1DigestExpected := "sha256:1a5634a500d044cfb6067558a3aed035e39992fe03406bdb743f336cd5837c6c"

	type testCase struct {
		uploadSignature       string
		uploadSignatureDigest string
		key                   string
		shouldError           bool
		description           string
		annotationrules       v1.SignatureAnnotations
	}

	testCases := []testCase{
		{
			uploadSignature:       sigNotOKPath,
			uploadSignatureDigest: sigNotOKDigestExpected,
			key:                   VALIDKEY1,
			shouldError:           true,
			description:           "should fail because key is valid but annotations do not match",
			annotationrules: v1.SignatureAnnotations{
				Match: map[string]string{
					"tag": "ok",
				},
			},
		},
		{
			uploadSignature:       sigNotOKPath,
			uploadSignatureDigest: sigNotOKDigestExpected,
			key:                   INVALIDKEY1,
			shouldError:           true,
			description:           "should fail because key is invalid",
			annotationrules: v1.SignatureAnnotations{
				Match: map[string]string{
					"tag": "ok",
				},
			},
		},
		{
			uploadSignature:       sigNotOK2Path,
			uploadSignatureDigest: sigNotOK2DigestExpected,
			key:                   VALIDKEY1,
			shouldError:           true,
			description:           "should fail because key is invalid but annotations are correct",
			annotationrules: v1.SignatureAnnotations{
				Match: map[string]string{
					"tag": "ok",
				},
			},
		},
		{
			uploadSignature:       sigOK1Path,
			uploadSignatureDigest: sigOK1DigestExpected,
			key:                   VALIDKEY1,
			shouldError:           false,
			description:           "should pass because key is valid and annotations match",
			annotationrules: v1.SignatureAnnotations{
				Match: map[string]string{
					"tag": "ok",
				},
			},
		},
	}

	// Set up a fake registry.
	s := httptest.NewServer(registry.New(registry.Logger(log.New(io.Discard, "", 0))))
	defer s.Close()
	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatal(err)
	}

	// Setup image
	imgRepo := fmt.Sprintf("%s/library/hello-world", u.Host)

	sigName := fmt.Sprintf("%s:%s", imgRepo, sigTag)

	sigRef, err := name.ParseReference(sigName)
	if err != nil {
		t.Fatal(err)
	}

	opts := VerifyOpts{
		ImageRef:           fmt.Sprintf("%s@%s", imgRepo, imgDigestExpected),
		SignatureAlgorithm: "sha256",
		NoCache:            true,
	}

	ref, err := name.ParseReference(opts.ImageRef)
	if err != nil {
		t.Fatal(err)
	}

	if err := pushOCIDir(imgPath, ref); err != nil {
		t.Fatal(err)
	}

	digest, err := crane.Digest(opts.ImageRef)
	if err != nil {
		t.Fatal(err)
	}

	if digest != imgDigestExpected {
		t.Fatalf("image digest mismatch: %s != %s", digest, imgDigestExpected)
	}

	// Loop through test cases
	for ti, tc := range testCases {
		if err := pushOCIDir(tc.uploadSignature, sigRef); err != nil {
			t.Fatal(err)
		}

		sigDigest, err := crane.Digest(sigName)
		if err != nil {
			t.Fatal(err)
		}

		if sigDigest != tc.uploadSignatureDigest {
			t.Fatalf("[%d] signature digest mismatch: %s != %s", ti, sigDigest, tc.uploadSignatureDigest)
		}

		opts.Key = tc.key
		opts.AnnotationRules = tc.annotationrules

		err = VerifySignature(context.Background(), nil, opts)
		if err != nil && !tc.shouldError {
			t.Fatalf("[%d] unexpected error: %v, but %s", ti, err, tc.description)
		}

		if err == nil && tc.shouldError {
			t.Fatalf("[%d] expected error but got none: %s", ti, tc.description)
		}
	}
}
