---
title: "acorn image rm"
---
## acorn image rm

Delete an Image

```
acorn image rm [IMAGE_NAME...] [flags]
```

### Examples

```
acorn image rm my-image
```

### Options

```
  -f, --force   Force Delete
  -h, --help    help for rm
```

### Options inherited from parent commands

```
  -a, --all                 Include untagged images
  -A, --all-projects        Use all known projects
  -c, --containers          Show containers for images
      --context string      Context to use in the resolved kubeconfig file
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding current project
      --namespace string    Namespace to work in resolved connection (default "acorn")
      --no-trunc            Don't truncate IDs
  -o, --output string       Output format (json, yaml, {{gotemplate}})
  -j, --project string      Project to work in
  -q, --quiet               Output only names
```

### SEE ALSO

* [acorn image](acorn_image.md)	 - Manage images

