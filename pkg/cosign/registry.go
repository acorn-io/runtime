package cosign

import (
	"fmt"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	ggcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sirupsen/logrus"
)

/* DISCLAIMER: Some parts of this code are copied from the crane package.
 * Source: github.com/google/go-containerregistry/pkg/crane
 * Original License below:
 * ------------------------------------------------------------
 * Copyright 2018 Google LLC All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 * ------------------------------------------------------------
 */

// SimpleDigest is an adaption of crane.Digest
//   - it returns the sha256 hash of the remote image at ref.
//   - removed: it does not support platform specific images (we don't need that here)
//   - added: it returns an error if the image is not found on first try with HEAD
//     (to lower the number of GET requests against potentially rate limited registries)
func SimpleDigest(ref name.Reference, opt ...crane.Option) (string, error) {
	o := makeOptions(opt...)
	desc, err := crane.Head(ref.Name(), opt...)
	if err != nil {
		if terr, ok := err.(*transport.Error); ok && terr.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("ref %s not found: %w", ref, terr)
		}
		logrus.Debugf("HEAD request failed for ref %s, falling back on GET: %v", ref, err)
		rdesc, err := remote.Get(ref, o.Remote...)
		if err != nil {
			return "", err
		}
		return rdesc.Digest.String(), nil
	}
	return desc.Digest.String(), nil
}

func makeOptions(opts ...crane.Option) crane.Options {
	opt := crane.Options{
		Remote: []remote.Option{
			remote.WithAuthFromKeychain(authn.DefaultKeychain),
		},
		Keychain: authn.DefaultKeychain,
	}
	for _, o := range opts {
		o(&opt)
	}
	return opt
}

func FindSignature(imageDigest name.Digest, opts ...remote.Option) (name.Tag, ggcrv1.Hash, error) {
	ociremoteOpts := []ociremote.Option{ociremote.WithRemoteOptions(opts...)}

	var tag name.Tag
	var hash ggcrv1.Hash
	var err error

	tag, err = ociremote.SignatureTag(imageDigest, ociremoteOpts...)
	if err != nil {
		return tag, hash, fmt.Errorf("failed to get signature tag: %w", err)
	}
	desc, err := remote.Head(tag, opts...) // HEAD request first to check if it exists (avoid rate limits)
	if err != nil {
		if terr, ok := err.(*transport.Error); ok && terr.StatusCode == http.StatusNotFound {
			logrus.Debugf("no signature found for image %s", imageDigest.String())
			return tag, hash, nil
		}
		return tag, hash, fmt.Errorf("error getting signature for image %s: %w", imageDigest.String(), err)
	}
	hash = desc.Digest

	return tag, hash, nil
}
