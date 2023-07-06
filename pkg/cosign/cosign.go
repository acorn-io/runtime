package cosign

import (
	"context"
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/acorn-io/baaah/pkg/merr"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/imagesystem"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	ggcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/cosign/v2/pkg/oci"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
	cosignature "github.com/sigstore/cosign/v2/pkg/signature"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/payload"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VerifyOpts struct {
	ImageRef           name.Digest
	SignatureRef       name.Reference
	Namespace          string
	AnnotationRules    v1.SignatureAnnotations
	Key                string
	SignatureAlgorithm string
	OciRemoteOpts      []ociremote.Option
	CraneOpts          []crane.Option
	NoCache            bool
	Verifiers          []signature.Verifier
}

// EnsureReferences will enrich the VerifyOpts with the image digest and signature reference.
// It's outsourced here, so we can ensure that it's used as few times as possible to reduce the number of potential
// GET requests to the registry which would count against potential rate limits.
func EnsureReferences(ctx context.Context, c client.Reader, img string, opts *VerifyOpts) error {
	if opts == nil {
		opts = &VerifyOpts{}
	}

	if opts.ImageRef.Identifier() == "" {
		// --- image name to digest hash
		imgRef, err := name.ParseReference(img)
		if err != nil {
			return fmt.Errorf("failed to parse image %s: %w", img, err)
		}

		// in the best case, we have a digest ref already, so we don't need to do any external request
		if imgDigest, ok := imgRef.(name.Digest); ok {
			opts.ImageRef = imgDigest
		} else {
			imgDigest, err := crane.Digest(imgRef.Name(), opts.CraneOpts...) // this uses HEAD to determine the digest, but falls back to GET if HEAD fails
			if err != nil {
				return fmt.Errorf("failed to resolve image digest: %w", err)
			}

			opts.ImageRef = imgRef.Context().Digest(imgDigest)
		}
	}

	if opts.SignatureRef == nil || opts.SignatureRef.Identifier() == "" {
		signatureRef, err := ensureSignatureArtifact(ctx, c, opts.Namespace, opts.ImageRef, opts.NoCache, opts.OciRemoteOpts, opts.CraneOpts)
		if err != nil {
			return err
		}
		opts.SignatureRef = signatureRef
	}

	return nil
}

func ensureSignatureArtifact(ctx context.Context, c client.Reader, namespace string, img name.Digest, noCache bool, ociRemoteOpts []ociremote.Option, craneOpts []crane.Option) (name.Reference, error) {
	// -- signature hash
	sigTag, err := ociremote.SignatureTag(img, ociRemoteOpts...) // we force imgRef to be a digest above, so this should *not* make a GET request to the registry
	if err != nil {
		return nil, fmt.Errorf("failed to get signature tag: %w", err)
	}

	sigDigest, err := SimpleDigest(sigTag, craneOpts...) // similar to crane.Digest, but fails if HEAD returns 404 Not Found
	if err != nil {
		var terr *transport.Error
		if ok := errors.As(err, &terr); ok && terr.StatusCode == http.StatusNotFound {
			// signature artifact not found -> that's an actual verification error
			cerr := cosign.NewVerificationError(fmt.Sprintf("signature verification failed: expected signature artifact %s not found", sigTag.Name()))
			cerr.(*cosign.VerificationError).SetErrorType(cosign.ErrNoSignaturesFoundType)
			return nil, cerr
		}
		return nil, fmt.Errorf("failed to get signature digest: %w", err)
	}

	sigRefToUse, err := name.ParseReference(sigTag.Name(), name.WeakValidation)
	if err != nil {
		return nil, fmt.Errorf("failed to parse signature reference: %w", err)
	}
	logrus.Debugf("Signature %s has digest: %s", sigRefToUse.Name(), sigDigest)

	if !noCache {
		internalRepo, _, err := imagesystem.GetInternalRepoForNamespace(ctx, c, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get internal repo for namespace %s: %w", namespace, err)
		}

		localSignatureArtifact := fmt.Sprintf("%s:%s", internalRepo, sigTag.Identifier())

		// --- check if we have the signature artifact locally, if not, copy it over from external registry
		mustPull := false
		localSigSHA, err := crane.Digest(localSignatureArtifact, craneOpts...) // this uses HEAD to determine the digest, but falls back to GET if HEAD fails
		if err != nil {
			var terr *transport.Error
			if ok := errors.As(err, &terr); ok && terr.StatusCode == http.StatusNotFound {
				logrus.Debugf("signature artifact %s not found locally, will try to pull it", localSignatureArtifact)
				mustPull = true
			} else {
				return nil, fmt.Errorf("failed to get local signature digest, cannot check if we have it cached locally: %w", err)
			}
		} else if localSigSHA != sigDigest {
			logrus.Debugf("Local signature digest %s does not match remote signature digest %s, will try to pull it", localSigSHA, sigDigest)
			mustPull = true
		}

		if mustPull {
			// --- pull signature artifact
			err := crane.Copy(sigTag.Name(), localSignatureArtifact, craneOpts...) // Pull (GET) counts against the rate limits, so this shouldn't be done too often
			if err != nil {
				return nil, fmt.Errorf("failed to copy signature artifact: %w", err)
			}
		}

		lname, err := name.ParseReference(localSignatureArtifact, name.WeakValidation)
		if err != nil {
			return nil, fmt.Errorf("failed to parse local signature artifact %s: %w", localSignatureArtifact, err)
		}

		sigRefToUse = lname

		logrus.Debugf("Checking if image %s is signed with %s (cache: %s)", img, sigTag, localSignatureArtifact)
	}

	return sigRefToUse, nil
}

// VerifySignature checks if the image is signed with the given key and if the annotations match the given rules
// This does a lot of image and image manifest juggling to fetch artifacts, digests, etc. from the registry, so we have to be
// careful to not do too many GET requests that count against registry rate limits (e.g. for Docker Hub).
// Crane uses HEAD (with GET as a fallback) wherever it can, so it's a good choice here e.g. for fetching digests.
func VerifySignature(ctx context.Context, opts VerifyOpts) error {
	sigs, err := ociremote.Signatures(opts.SignatureRef, opts.OciRemoteOpts...) // this runs against our internal registry, so it should not count against the rate limits
	if err != nil {
		return fmt.Errorf("failed to get signatures: %w", err)
	}

	imgDigestHash, err := ggcrv1.NewHash(opts.ImageRef.DigestStr())
	if err != nil {
		return err
	}

	// --- cosign verifier options

	cosignOpts := &cosign.CheckOpts{
		Annotations:        map[string]interface{}{},
		ClaimVerifier:      cosign.SimpleClaimVerifier,
		RegistryClientOpts: opts.OciRemoteOpts,
		IgnoreTlog:         true,
	}

	if opts.Verifiers == nil {
		opts.Verifiers = []signature.Verifier{}
	}

	// --- parse key
	if opts.Key != "" {
		verifiers, err := LoadVerifiers(ctx, opts.Key, opts.SignatureAlgorithm)
		if err != nil {
			return fmt.Errorf("failed to load key: %w", err)
		}
		opts.Verifiers = append(opts.Verifiers, verifiers...)
	}

	verified := false
	var errs []error
	for _, v := range opts.Verifiers {
		cosignOpts.SigVerifier = v
		err := verifySignature(ctx, sigs, imgDigestHash, opts, cosignOpts)
		if err == nil {
			verified = true
			break
		}
		errs = append(errs, err)
	}

	if !verified {
		err := cosign.NewVerificationError("failed to find valid signature for %s matching given identity and annotations using %d loaded verifiers/keys", opts.ImageRef.String(), len(opts.Verifiers))
		err.(*cosign.VerificationError).SetErrorType(cosign.ErrNoMatchingSignaturesType)
		logrus.Debugf("%s: %v", err, merr.NewErrors(errs...))
		return err
	}

	return nil
}

func verifySignature(ctx context.Context, sigs oci.Signatures, imgDigestHash ggcrv1.Hash, opts VerifyOpts, cosignOpts *cosign.CheckOpts) error {
	// --- get and verify signatures
	signatures, bundlesVerified, err := verifySignatures(ctx, sigs, imgDigestHash, cosignOpts)
	if err != nil {
		if _, ok := err.(*cosign.VerificationError); ok {
			return err
		}
		return fmt.Errorf("failed to verify image signatures: %w", err)
	}

	logrus.Debugf("image %s: %d signatures verified (bundle verified: %v)", opts.ImageRef.Name(), len(signatures), bundlesVerified)

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
	logrus.Debugf("image %s: Annotations (%+v) verified", opts.ImageRef.Name(), opts.AnnotationRules)

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
		var verr error
		bundleVerified, verr = cosign.VerifyImageSignature(ctx, sig, h, co)
		if verr != nil {
			validationErrs = append(validationErrs, verr.Error())
			continue
		}

		checkedSignatures = append(checkedSignatures, sig)
	}
	if len(checkedSignatures) == 0 {
		cerr := cosign.NewVerificationError(cosign.ErrNoMatchingSignaturesMessage)
		cerr.(*cosign.VerificationError).SetErrorType(cosign.ErrNoMatchingSignaturesType)
		return nil, false, fmt.Errorf("%w:\n%s", cerr, strings.Join(validationErrs, "\n "))
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

func LoadVerifiers(ctx context.Context, keyRef string, algorithm string) (verifiers []signature.Verifier, err error) {
	if PubkeyPrefixPattern.MatchString(strings.TrimSpace(keyRef)) {
		// no scheme, inline PEM
		v, err := decodePEM([]byte(keyRef), algorithms[algorithm])
		if err != nil {
			return nil, fmt.Errorf("failed to load public key from PEM: %w", err)
		}
		verifiers = append(verifiers, v)
	} else if strings.HasPrefix(keyRef, "ssh-") {
		keyData := strings.Fields(keyRef)[1]
		parsedCryptoKey, err := ParseSSHPublicKey(keyData)
		if err != nil {
			return nil, err
		}
		v, err := signature.LoadVerifier(parsedCryptoKey, algorithms[algorithm])
		if err != nil {
			return nil, fmt.Errorf("failed to load public key from SSH: %w", err)
		}
		verifiers = append(verifiers, v)
	} else if strings.HasPrefix(keyRef, "-----BEGIN") {
		key, err := ParseSSHPublicKey(keyRef)
		if err != nil {
			return nil, err
		}
		logrus.Infof("Parsed key: %#v", key)
		v, err := signature.LoadVerifier(key, algorithms[algorithm])
		if err != nil {
			return nil, fmt.Errorf("failed to load public key from PEM: %w", err)
		}
		verifiers = append(verifiers, v)
	} else if strings.HasPrefix(keyRef, "ac://") {
		// Acorn Manager
		acKeys, err := getAcornPublicKeys(strings.TrimPrefix(keyRef, "ac://"))
		if err != nil {
			return nil, fmt.Errorf("failed to load public keys from Acorn Manager: %w", err)
		}

		var acVerifiers []signature.Verifier
		for _, key := range acKeys {
			v, err := LoadVerifiers(ctx, key.Key, algorithm)
			if err != nil {
				logrus.Debugf("failed to load public key from Acorn Manager for %s: %v", keyRef, err)
				continue
			}
			acVerifiers = append(acVerifiers, v...)
		}

		if len(acVerifiers) == 0 {
			return nil, fmt.Errorf("no supported public keys found in Acorn Manager for %s", keyRef)
		}

		verifiers = append(verifiers, acVerifiers...)
	} else if strings.HasPrefix(keyRef, "gh://") {
		// gh://
		ghKeys, err := getGitHubPublicKeys(strings.TrimPrefix(keyRef, "gh://"))
		if err != nil {
			return nil, fmt.Errorf("failed to load public keys from GitHub: %w", err)
		}

		var ghVerifiers []signature.Verifier

		for _, key := range ghKeys {
			v, err := LoadVerifiers(ctx, key.Key, algorithm)
			if err != nil {
				logrus.Debugf("failed to load verifier for public key from GitHub (type %T): %v", key, err)
				continue
			}
			ghVerifiers = append(ghVerifiers, v...)
		}

		if len(ghVerifiers) == 0 {
			return nil, fmt.Errorf("no supported public keys found in GitHub for %s", keyRef)
		}

		verifiers = append(verifiers, ghVerifiers...)
	} else {
		// schemes: k8s://, pkcs11://, gitlab://
		v, err := cosignature.PublicKeyFromKeyRef(ctx, keyRef)
		if err != nil {
			return nil, fmt.Errorf("failed to load public key from %s: %w", keyRef, err)
		}
		verifiers = append(verifiers, v)
	}

	if len(verifiers) == 0 {
		return nil, fmt.Errorf("error: no public keys loaded from %s", keyRef)
	}

	return
}
