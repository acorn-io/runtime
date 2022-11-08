---
title: "acorn image rm"
---
## acorn image rm

Delete a image

```
acorn image rm [IMAGE_NAME...] [flags]
```

### Examples

```

acorn image rm my-image
```

### Options

```
  -h, --help   help for rm
```

### Options inherited from parent commands

```
  -A, --all-namespaces      Namespace to work in
      --context string      Context to use in the kubeconfig file
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Location of a kubeconfig file
      --namespace string    Namespace to work in (default "acorn")
  -o, --output string       Output format (json, yaml, {{gotemplate}})
  -q, --quiet               Output only names
```

### SEE ALSO

* [acorn image](acorn_image.md)	 - Manage images

