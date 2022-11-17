---
title: "acorn image"
---
## acorn image

Manage images

```
acorn image [flags] [APP_NAME...]
```

### Examples

```

acorn images
```

### Options

```
  -a, --all             Include untagged images
  -c, --containers      Show containers for images
  -h, --help            help for image
      --no-trunc        Don't truncate IDs
  -o, --output string   Output format (json, yaml, {{gotemplate}})
  -q, --quiet           Output only names
```

### Options inherited from parent commands

```
  -A, --all-namespaces      Namespace to work in
      --context string      Context to use in the kubeconfig file
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Location of a kubeconfig file
      --namespace string    Namespace to work in (default "acorn")
```

### SEE ALSO

* [acorn](acorn.md)	 - 
* [acorn image rm](acorn_image_rm.md)	 - Delete an Image

