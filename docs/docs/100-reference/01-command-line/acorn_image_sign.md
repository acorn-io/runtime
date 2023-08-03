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
  -a, --annotation strings   Annotations to add to the signature
  -h, --help                 help for sign
  -k, --key string           Key to use for signing
```

### Options inherited from parent commands

```
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding the default context
  -j, --project string      Project to work in
```

### SEE ALSO

* [acorn image](acorn_image.md)	 - Manage images

