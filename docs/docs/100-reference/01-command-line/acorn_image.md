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
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding the default context
  -j, --project string      Project to work in
```

### SEE ALSO

* [acorn](acorn.md)	 - 
* [acorn image copy](acorn_image_copy.md)	 - Copy Acorn images between registries
* [acorn image details](acorn_image_details.md)	 - Show details of an Image
* [acorn image rm](acorn_image_rm.md)	 - Delete an Image

