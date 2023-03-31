package cosign

import (
	"context"
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/imagesystem"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	ggcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/cosign/pkg/oci"
	ociremote "github.com/sigstore/cosign/pkg/oci/remote"
	"github.com/sigstore/cosign/pkg/oci/static"
	cosignature "github.com/sigstore/cosign/pkg/signature"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/payload"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VerifyOpts struct {
	ImageRef           string
	Namespace          string
	AnnotationRules    v1.SignatureAnnotations
	Key                string
	SignatureAlgorithm string
	OciRemoteOpts      []ociremote.Option
	CraneOpts          []crane.Option
	NoCache            bool
}

// VerifySignature checks if the image is signed with the given key and if the annotations match the given rules
// This does a lot of image and image manifest juggling to fetch artifacts, digests, etc. from the registry, so we have to be
// careful to not do too many GET requests that count against registry rate limits (e.g. for DockerHub).
// Crane uses HEAD (with GET as a fallback) wherever it can, so it's a good choice here e.g. for fetching digests.
func VerifySignature(ctx context.Context, c client.Reader, opts VerifyOpts) error {
	// --- image name to digest hash
	imgRef, err := name.ParseReference(opts.ImageRef)
	if err != nil {
		return fmt.Errorf("failed to parse image %s: %w", opts.ImageRef, err)
	}

	imgDigest, err := crane.Digest(imgRef.Name(), opts.CraneOpts...) // this uses HEAD to determine the digest, but falls back to GET if HEAD fails
	if err != nil {
		return fmt.Errorf("failed to resolve image digest: %w", err)
	}

	imgRef = imgRef.Context().Digest(imgDigest)

	imgDigestHash, err := ggcrv1.NewHash(imgDigest)
	if err != nil {
		return err
	}

	// --- cosign verifier options

	cosignOpts := &cosign.CheckOpts{
		Annotations:        map[string]interface{}{},
		ClaimVerifier:      cosign.SimpleClaimVerifier,
		RegistryClientOpts: opts.OciRemoteOpts,
	}

	// --- parse key
	if opts.Key != "" {
		verifier, err := LoadKey(ctx, opts.Key, opts.SignatureAlgorithm)
		if err != nil {
			return fmt.Errorf("failed to load key: %w", err)
		}
		cosignOpts.SigVerifier = verifier
	}

	// -- signature hash
	sigTag, err := ociremote.SignatureTag(imgRef, opts.OciRemoteOpts...) // we force imgRef to be a digest above, so this should *not* make a GET request to the registry
	if err != nil {
		return fmt.Errorf("failed to get signature tag: %w", err)
	}

	sigDigest, err := crane.Digest(sigTag.Name(), opts.CraneOpts...) // this uses HEAD to determine the digest, but falls back to GET if HEAD fails
	if err != nil {
		if terr, ok := err.(*transport.Error); ok && terr.StatusCode == http.StatusNotFound {
			// signature artifact not found -> that's an actual verification error
			return fmt.Errorf("%w: expected signature artifact %s not found", cosign.ErrNoMatchingSignatures, sigTag.Name())
		}
		return fmt.Errorf("failed to get signature digest: %w", err)
	}

	sigRefToUse, err := name.ParseReference(sigTag.Name(), name.WeakValidation)
	if err != nil {
		return fmt.Errorf("failed to parse signature reference: %w", err)
	}
	logrus.Debugf("Signature %s has digest: %s", sigRefToUse.Name(), sigDigest)

	if !opts.NoCache {
		internalRepo, _, err := imagesystem.GetInternalRepoForNamespace(ctx, c, opts.Namespace)
		if err != nil {
			return fmt.Errorf("failed to get internal repo for namespace %s: %w", opts.Namespace, err)
		}

		localSignatureArtifact := fmt.Sprintf("%s:%s", internalRepo, sigTag.Identifier())

		// --- check if we have the signature artifact locally, if not, copy it over from external registry
		mustPull := false
		localSigSHA, err := crane.Digest(localSignatureArtifact, opts.CraneOpts...) // this uses HEAD to determine the digest, but falls back to GET if HEAD fails
		if err != nil {
			var terr *transport.Error
			if ok := errors.As(err, &terr); ok && terr.StatusCode == http.StatusNotFound {
				logrus.Debugf("signature artifact %s not found locally, will try to pull it", localSignatureArtifact)
				mustPull = true
			} else {
				return fmt.Errorf("failed to get local signature digest, cannot check if we have it cached locally: %w", err)
			}
		} else if localSigSHA != sigDigest {
			logrus.Debugf("Local signature digest %s does not match remote signature digest %s, will try to pull it", localSigSHA, sigDigest)
			mustPull = true
		}

		if mustPull {
			// --- pull signature artifact
			err := crane.Copy(sigTag.Name(), localSignatureArtifact, opts.CraneOpts...) // Pull (GET) counts against the rate limits, so this shouldn't be done too often
			if err != nil {
				return fmt.Errorf("failed to copy signature artifact: %w", err)
			}
		}

		lname, err := name.ParseReference(localSignatureArtifact, name.WeakValidation)
		if err != nil {
			return fmt.Errorf("failed to parse local signature artifact %s: %w", localSignatureArtifact, err)
		}

		sigRefToUse = lname

		logrus.Debugf("Checking if image %s is signed with %s (cache: %s)", imgRef, sigTag, localSignatureArtifact)
	}

	sigs, err := ociremote.Signatures(sigRefToUse, ociremote.WithRemoteOptions(remote.WithAuthFromKeychain(authn.DefaultKeychain))) // this runs against our internal registry, so it should not count against the rate limits
	if err != nil {
		return fmt.Errorf("failed to get signatures: %w", err)
	}

	// --- get and verify signatures
	signatures, bundlesVerified, err := verifySignatures(ctx, sigs, imgDigestHash, cosignOpts)
	if err != nil {
		if _, ok := err.(*cosign.VerificationError); ok {
			return err
		}
		return fmt.Errorf("failed to verify image signatures: %w", err)
	}

	logrus.Debugf("image %s: %d signatures verified (bundle verified: %v)", imgRef, len(signatures), bundlesVerified)

	// --- extract payloads for subsequent checks
	payloads, err := extractPayload(signatures)
	if err != nil {
		return fmt.Errorf("failed to extract payload: %w", err)
	}

	// --- check annotations
	if err := checkAnnotations(payloads, opts.AnnotationRules); err != nil {
		if _, ok := err.(*cosign.VerificationError); ok {
			return err
		}
		return fmt.Errorf("failed to check annotations: %w", err)
	}
	logrus.Debugf("image %s: Annotations (%+v) verified", imgRef, opts.AnnotationRules)

	return nil
}

func decodePEM(raw []byte, signatureAlgorithm crypto.Hash) (signature.Verifier, error) {
	// PEM encoded file.
	pubKey, err := cryptoutils.UnmarshalPEMToPublicKey(raw)
	if err != nil {
		return nil, fmt.Errorf("pem to public key: %w", err)
	}

	return signature.LoadVerifier(pubKey, signatureAlgorithm)
}

var ErrAnnotationsUnmatched = cosign.NewVerificationError("annotations unmatched")

func checkAnnotations(payloads []payload.SimpleContainerImage, annotationRule v1.SignatureAnnotations) error {
	sel, err := annotationRule.AsSelector()
	if err != nil {
		return fmt.Errorf("failed to parse annotation rule: %w", err)
	}

	if sel.Empty() {
		return nil
	}

	kvLists := map[string][]string{}
	for _, p := range payloads {
		for k, v := range p.Optional {
			if v != nil {
				kvLists[k] = append(kvLists[k], fmt.Sprint(v))
			}
		}
	}

	labelMaps := generateVariations(kvLists) // alternatively we can re-write the label matching logic to work with a map[string][]string
	logrus.Debugf("Checking against %d generated label maps: %+v", len(labelMaps), labelMaps)

	for _, labelMap := range labelMaps {
		if sel.Matches(labels.Set(labelMap)) {
			return nil
		}
	}

	logrus.Debugf("No label map variation matched the selector %+v", sel)

	return ErrAnnotationsUnmatched
}

func generateVariations(input map[string][]string) []map[string]string {
	// Count the number of variations
	count := 1
	for _, values := range input {
		count *= len(values)
	}

	// Initialize the output slice
	output := make([]map[string]string, count)

	// Generate variations
	for i := 0; i < count; i++ {
		output[i] = make(map[string]string)
		j := 1
		for key, values := range input {
			output[i][key] = values[(i/j)%len(values)]
			j *= len(values)
		}
	}

	return output
}

func verifySignatures(ctx context.Context, sigs oci.Signatures, h ggcrv1.Hash, co *cosign.CheckOpts) (checkedSignatures []oci.Signature, bundleVerified bool, err error) {
	sl, err := sigs.Get()
	if err != nil {
		return nil, false, err
	}

	validationErrs := []string{}

	for _, sig := range sl {
		sig, err := static.Copy(sig)
		if err != nil {
			validationErrs = append(validationErrs, err.Error())
			continue
		}
		verified, err := cosign.VerifyImageSignature(ctx, sig, h, co)
		bundleVerified = bundleVerified || verified
		if err != nil {
			validationErrs = append(validationErrs, err.Error())
			continue
		}

		checkedSignatures = append(checkedSignatures, sig)
	}
	if len(checkedSignatures) == 0 {
		return nil, false, fmt.Errorf("%w:\n%s", cosign.ErrNoMatchingSignatures, strings.Join(validationErrs, "\n "))
	}
	return checkedSignatures, bundleVerified, nil
}

func extractPayload(verified []oci.Signature) ([]payload.SimpleContainerImage, error) {
	var sigPayloads []payload.SimpleContainerImage
	for _, sig := range verified {
		pld, err := sig.Payload()
		if err != nil {
			return nil, fmt.Errorf("failed to get payload: %w", err)
		}

		sci := payload.SimpleContainerImage{}
		if err := json.Unmarshal(pld, &sci); err != nil {
			return nil, fmt.Errorf("error decoding the payload: %w", err)
		}

		sigPayloads = append(sigPayloads, sci)
	}
	return sigPayloads, nil
}

var algorithms = map[string]crypto.Hash{
	"":       crypto.SHA256,
	"sha256": crypto.SHA256,
	"sha512": crypto.SHA512,
}

func LoadKey(ctx context.Context, keyRef string, algorithm string) (verifier signature.Verifier, err error) {
	if strings.HasPrefix(strings.TrimSpace(keyRef), "-----BEGIN PUBLIC KEY-----") {
		// no scheme, inline PEM
		verifier, err = decodePEM([]byte(keyRef), algorithms[algorithm])
		if err != nil {
			return nil, fmt.Errorf("failed to load public key from PEM: %w", err)
		}
		// TODO: add github
	} else {
		// schemes: k8s://, pkcs11://, gitlab://
		verifier, err = cosignature.PublicKeyFromKeyRef(ctx, keyRef)
		if err != nil {
			return nil, fmt.Errorf("failed to load public key from %s: %w", keyRef, err)
		}
	}

	return
}
