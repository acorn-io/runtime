# [Alpha Feature] ImageAllowRules (IARs)

ImageAllowRules are an alpha-feature of Acorn, currently hidden behind a feature flag.
To enable it in your Acorn installation, use `acorn isntall --features image-allow-rules=true` (**Beware**: first read the rest of this page before enabling it, as it's quite disruptive).

## How ImageAllowRules work

The principle behind IARs is to make your cluster more secure.
In fact, secure by default, once it's enabled. No additional policies needed.
If you enable this feature, you won't be able to deploy any Acorn image anymore without allowing it.
To do that, you have to create an `ImageAllowRule` resource.
If the app image is allowed by a single IAR in your project, it's good to run.

**Note**: Removing an IAR won't stop your running app, but will update the `image-allowed` status condition on the app.

## What makes up an ImageAllowRule

Currently, IARs have two parts:

1. The `images` scope (must-have), denoting, what images the rule applies to. It uses the same syntax as the auto-upgrade pattern. Examples below.
2. The `signatures` rules (optional) define a set of image signatures and annotations on those signatures to make sure that an image was actually approved by someone or something, e.g. by your QA team. We're using [sigstore/cosign](https://docs.sigstore.dev/cosign/installation/) for everything related to signatures.

## Example

```yaml
apiVersion: api.acorn.io/v1
kind: ImageAllowRule
metadata:
  name: example-iar
  namespace: acorn # your project namespace
images:
  - ghcr.io/** # ** matches everything, * matches a single path item, # matches a number
signatures:
  rules:
    - signedBy:
        anyOf: # one match is good enough
          - |
            -----BEGIN PUBLIC KEY-----
            MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEo9QMl0ilxrBNFqOpifkhmKVZ14D8
            cUSzwOtALU9owM2ZRzE55OP4je2y9sTVvlNr59eZQ/Q4gsxHfo4EETEuog==
            -----END PUBLIC KEY-----
        allOf: [] # all signatures required
      annotations: # those annotations have to be present on all signatures
        match: # simple key-value pairs
          qa: approved
        expressions: # just like Kubernetes label selectors
          - key: tests
            operator: In # In, NotIn, Exists, DoesNotExist
            values:
              - passed
              - ok
```

## About Signatures

To sign an image, you can use [sigstore/cosign](https://docs.sigstore.dev/cosign/installation/) via the CLI.
You download or build an Acorn image, you sign it with cosign and optionally annotate the signature, then you upload it to some OCI registry.
Now when you try to run that in a protected cluster and the image is in scope of an IAR, Acorn will check the signature, make sure that it matches the provided public keys and that matching signatures also have the required annotations.
Only if all of this is true, we let the image pass.

### Walkthrough

Here's a full walkthrough to use Acorn with the ImageAllowRules feature in a fresh installation and with cosign signatures.
Please note that the exact output may be different for you, especially depending on the version of cosign you use.

```bash
# 1. Create a fresh cluster and install Acorn - doesn't have to be k3d, you may also update your existing installation
$ k3d cluster create acorn
...

$ acorn install --features image-allow-rules=true
...

# 2. Pull some Acorn image and push it to another registry that you have push access to (Alternatively, build it from an Acornfile)
$ acorn pull ghcr.io/acorn-io/library/hello-world:latest
$ acorn tag ghcr.io/acorn-io/library/hello-world:latest my.registry.local/acorn/hello-world:latest
$ acorn push my.registry.local/acorn/hello-world:latest

# 2.1 Faster using crane: 
$ crane copy ghcr.io/acorn-io/library/hello-world:latest my.registry.local/acorn/hello-world:latest
...

# 3. Get the digest
$ crane digest my.registry.local/acorn/hello-world:latest
sha256:1a6c64d2ccd0bb035f9c8196d3bfe72a7fdbddc4530dfcb3ab2a0ab8afb57eeb

# 4. Generate a keypair if you don't have one already
$ cosign generate-key-pair
Enter password for private key: 
Enter password for private key again: 
WARNING: File cosign.key already exists. Overwrite?
Are you sure you would like to continue? [y/N] y
Private key written to cosign.key
Public key written to cosign.pub

# 5. Sign the image with the newly generated key and an annotation that says `tag=notok`
$ cosign sign --key cosign.key -a tag=notok my.registry.local/acorn/hello-world@sha256:1a6c64d2ccd0bb035f9c8196d3bfe72a7fdbddc4530dfcb3ab2a0ab8afb57eeb
Enter password for private key: 

 Note that there may be personally identifiable information associated with this signed artifact.
 This may include the email address associated with the account with which you authenticate.
 This information will be used for signing this artifact and will be stored in public transparency logs and cannot be removed later.

By typing 'y', you attest that you grant (or have permission to grant) and agree to have this information stored permanently in transparency logs.
Are you sure you would like to continue? [y/N] y
tlog entry created with index: 15549911
Pushing signature to: my.registry.local/acorn/hello-world
 
# 6. Deploy a cluster-level image allow rule that will deny this image
$ cat << EOF | kubectl apply -f -           
pipe heredoc> apiVersion: api.acorn.io/v1
kind: ImageAllowRule
metadata:
  name: testrule
  namespace: acorn
images:
  - my.registry.local/**
signatures:
  rules:
    - signedBy:
        anyOf:
          - |
            -----BEGIN PUBLIC KEY-----
            !!! Put your Public Key here !!!
            -----END PUBLIC KEY-----
      annotations:
        match:
          tag: ok
EOF
imageallowrule.api.acorn.io/testrule configured

# 7. Try to run the image -> it should fail (because the annotation is wrong)

$ acorn run my.registry.local/acorn/hello-world:latest        
  •  WARNING:  This application would like to use the image 'my.registry.local/acorn/hello-world:latest'.
                 This image is not trusted by any image allow rules in this project.
                 This could be VERY DANGEROUS to the cluster if you do not trust this
                 application. If you are unsure say no.

? Do you want to allow this app to use this (POTENTIALLY DANGEROUS) image?  [Use arrows to move, type to filter]
> NO
  yes (this tag only)
  repository (all images in this repository)
  registry (all images in this registry)
  all (all images out there)

# Here, as an admin, you get to choose to have Acorn automatically generate an IAR for you to allow this image (without signatures)
# Repeating Step 5 with `-a tag=notok` and then continuing with steps 6 and 7, should make it work
...
```

## No need for YAML

As you have seen in the last section, Acorn also prompts admins to allow an image that is not yet allowed to run. That's quite basic and will create an ImageAllowRule with only the `images` scope populated, no signatures required.
