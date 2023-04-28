---
title: "acorn image"
---
## acorn image

Manage images

```
acorn image [flags] [IMAGE_REPO:TAG|IMAGE_ID]
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
  -A, --all-projects        Use all known projects
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding current project
  -j, --project string      Project to work in
```

### SEE ALSO

* [acorn](acorn.md)	 - 
* [acorn image detail](acorn_image_detail.md)	 - Show details of an Image
* [acorn image rm](acorn_image_rm.md)	 - Delete an Image

