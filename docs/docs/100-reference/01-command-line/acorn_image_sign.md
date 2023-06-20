---
title: "acorn image sign"
---
## acorn image sign

Sign an Image

```
acorn image sign IMAGE_NAME [flags]
```

### Examples

```
acorn image sign my-image --key ./my-key
```

### Options

```
  -a, --annotations strings           Annotations to add to the signature
  -h, --help                          help for sign
  -k, --key string                    Key to use for signing (default "./cosign.key")
  -p, --push                          Push the signature to the signature repository
  -r, --signature-repository string   Repository to push the signature to
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

