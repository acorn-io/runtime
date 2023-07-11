---
title: "acorn image verify"
---
## acorn image verify

Verify Image Signatures

```
acorn image verify IMAGE_NAME [flags]
```

### Examples

```
# Verify using a locally stored public key file
acorn image verify my-image --key ./my-key.pub

# Verify using a public key belonging to a GitHub Identity
acorn image verify my-image --key gh://ibuildthecloud

# Verify using a public key belonging to an Acorn Manager Identity
acorn image verify my-image --key ac://ibuildthecloud

```

### Options

```
  -a, --annotations strings   Annotations to check for in the signature
  -h, --help                  help for verify
  -k, --key string            Key to use for verifying (default "./cosign.pub")
```

### Options inherited from parent commands

```
  -A, --all-projects        Use all known projects
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding current project
  -j, --project string      Project to work in
```

### SEE ALSO

* [acorn image](acorn_image.md)	 - Manage images

