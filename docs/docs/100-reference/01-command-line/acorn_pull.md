---
title: "acorn pull"
---
## acorn pull

Pull an image from a remote registry

```
acorn pull [flags] IMAGE
```

### Options

```
  -a, --annotation strings   Annotations to check for during verification
  -h, --help                 help for pull
  -k, --key string           Key to use for verifying (default "./cosign.pub")
  -v, --verify               Verify the image signature BEFORE pulling and only pull on success
```

### Options inherited from parent commands

```
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding the default context
  -j, --project string      Project to work in
```

### SEE ALSO

* [acorn](acorn.md)	 - 

