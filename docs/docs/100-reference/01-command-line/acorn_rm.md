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

acorn rm APP_NAME
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
  -A, --all-projects        Use all known projects
      --context string      Context to use in the resolved kubeconfig file
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding current project
      --namespace string    Namespace to work in resolved connection (default "acorn")
  -j, --project string      Project to work in
```

### SEE ALSO

* [acorn](acorn.md)	 - 

