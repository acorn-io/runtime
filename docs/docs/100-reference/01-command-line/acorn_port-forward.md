---
title: "acorn port-forward"
---
## acorn port-forward

Forward a container port locally

### Synopsis

Forward a container port locally

```
acorn port-forward [flags] ACORN_NAME|CONTAINER_NAME PORT
```

### Options

```
      --address string     The IP address to listen on (default "127.0.0.1")
  -c, --container string   Name of container to port forward into
  -h, --help               help for port-forward
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

