---
title: "acorn volume rm"
---
## acorn volume rm

Delete a volume

```
acorn volume rm [VOLUME_NAME...] [flags]
```

### Examples

```
acorn volume rm my-volume
```

### Options

```
  -h, --help   help for rm
```

### Options inherited from parent commands

```
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding current project
  -o, --output string       Output format (json, yaml, {{gotemplate}})
  -j, --project string      Project to work in
  -q, --quiet               Output only names
```

### SEE ALSO

* [acorn volume](acorn_volume.md)	 - Manage volumes

