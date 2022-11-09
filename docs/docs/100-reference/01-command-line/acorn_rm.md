---
title: "acorn rm"
---
## acorn rm

Delete an app, container, secret or volume

```
acorn rm [flags] [APP_NAME...]
```

### Examples

```

acorn rm
acorn rm -t volume,container APP_NAME
```

### Options

```
  -a, --all            Delete all types
  -f, --force          Force Delete
  -h, --help           help for rm
  -t, --type strings   Delete by type (container,app,volume,secret or c,a,v,s)
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

