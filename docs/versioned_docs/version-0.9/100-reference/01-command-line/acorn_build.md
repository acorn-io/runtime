---
title: "acorn build"
---
## acorn build

Build an app from a Acornfile file

### Synopsis

Build all dependent container and app images from your Acornfile file

```
acorn build [flags] DIRECTORY
```

### Examples

```

# Build from Acornfile file in the local directory
acorn build .
```

### Options

```
      --args-file string   Default args to apply to the build (default ".build-args.acorn")
  -f, --file string        Name of the build file (default "DIRECTORY/Acornfile")
  -h, --help               help for build
  -p, --platform strings   Target platforms (form os/arch[/variant][:osversion] example linux/amd64)
      --push               Push image after build
  -t, --tag strings        Apply a tag to the final build
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

* [acorn](acorn.md)	 - 

