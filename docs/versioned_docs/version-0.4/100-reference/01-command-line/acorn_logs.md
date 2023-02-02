---
title: "acorn logs"
---
## acorn logs

Log all pods from app

```
acorn logs [flags] APP_NAME|CONTAINER_NAME
```

### Options

```
  -f, --follow         Follow log output
  -h, --help           help for logs
  -s, --since string   Show logs since timestamp (e.g. 42m for 42 minutes)
  -n, --tail int       Number of lines in log output
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

