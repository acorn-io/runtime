---
title: "acorn rm"
---
## acorn rm

Delete an app, container, or volume

```
acorn rm [flags] [APP_NAME|VOL_NAME...]
```

### Examples

```

acorn rm
acorn rm -v some-volume
```

### Options

```
  -a, --all          Delete all types
  -c, --containers   Delete apps/containers
  -h, --help         help for rm
  -i, --images       Delete images/tags
  -s, --secrets      Delete secrets
  -v, --volumes      Delete volumes
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

