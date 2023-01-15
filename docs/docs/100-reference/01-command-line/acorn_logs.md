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

