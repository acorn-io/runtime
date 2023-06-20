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
acorn image verify my-image --key ./my-key.pub
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

