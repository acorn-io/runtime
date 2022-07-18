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
  -f, --file string        Name of the build file (default "DIRECTORY/Acornfile")
  -h, --help               help for build
  -p, --platform strings   Target platforms (form os/arch[/variant][:osversion] example linux/amd64)
      --profile strings    Profile to assign default values
  -t, --tag strings        Apply a tag to the final build
```

### Options inherited from parent commands

```
  -A, --all-namespaces      Namespace to work in
      --context string      Context to use in the kubeconfig file
      --kubeconfig string   Location of a kubeconfig file
      --namespace string    Namespace to work in (default "acorn")
```

### SEE ALSO

* [acorn](acorn.md)	 - 

