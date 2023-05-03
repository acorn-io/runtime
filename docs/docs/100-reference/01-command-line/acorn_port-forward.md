---
title: "acorn port-forward"
---
## acorn port-forward

Forward a container port locally

### Synopsis

Forward a container port locally

```
acorn port-forward [flags] APP_NAME|CONTAINER_NAME PORT
```

### Options

```
      --address string     The IP address to listen on (default "127.0.0.1")
  -c, --container string   Name of container to port forward into
  -h, --help               help for port-forward
```

### Options inherited from parent commands

```
  -A, --all-projects        Use all known projects
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding current project
  -j, --project string      Project to work in
```

### SEE ALSO

* [acorn](acorn.md)	 - 

