---
title: "acorn image copy"
---
## acorn image copy

Copy Acorn images between registries

```
acorn image copy [flags] SOURCE DESTINATION

  This command copies Acorn images between remote image registries.
  It does not interact with images stored in the Acorn internal registry, or with the Acorn API in any way.
```

### Examples

```
  # Copy an image from Docker Hub to GHCR:
    acorn copy docker.io/<username>/myimage:v1 ghcr.io/<username>/myimage:v1

  # Copy the 'main' tag on an image to the 'prod' tag on the same image, and overwrite if it already exists:
    acorn copy docker.io/<username>/myimage:main prod --force

  # Copy all tags on a particular image repo in Docker Hub to GHCR:
    acorn copy --all-tags docker.io/<username>/myimage ghcr.io/<username>/myimage
```

### Options

```
  -a, --all-tags   Copy all tags of the image
  -f, --force      Overwrite the destination image if it already exists
  -h, --help       help for copy
```

### Options inherited from parent commands

```
      --config-file string   Path of the acorn config file to use
      --debug                Enable debug logging
      --debug-level int      Debug log level (valid 0-9) (default 7)
      --kubeconfig string    Explicitly use kubeconfig file, overriding the default context
  -j, --project string       Project to work in
```

### SEE ALSO

* [acorn image](acorn_image.md)	 - Manage images

