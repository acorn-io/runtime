---
title: "acorn logs"
---
## acorn logs

Log all workloads from an app

```
acorn logs [flags] [ACORN_NAME|CONTAINER_REPLICA_NAME]
```

### Options

```
  -c, --container string   Container name or Job name within app to follow
  -f, --follow             Follow log output
  -h, --help               help for logs
  -s, --since string       Show logs since timestamp (e.g. 42m for 42 minutes)
  -n, --tail int           Number of lines in log output
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

