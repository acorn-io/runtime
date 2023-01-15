---
title: "acorn logout"
---
## acorn logout

Remove registry credentials

```
acorn logout [flags] [SERVER_ADDRESS]
```

### Examples

```

acorn logout ghcr.io
```

### Options

```
  -h, --help            help for logout
  -l, --local-storage   Delete locally stored credential (not remotely stored)
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

