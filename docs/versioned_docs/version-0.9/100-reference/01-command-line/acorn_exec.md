---
title: "acorn exec"
---
## acorn exec

Run a command in a container

### Synopsis

Run a command in a container

```
acorn exec [flags] ACORN_NAME|CONTAINER_NAME CMD
```

### Options

```
  -c, --container string     Name of container to exec into
  -d, --debug-image string   Use image as container root for command
  -h, --help                 help for exec
  -i, --interactive          Not used
  -t, --tty                  Not used
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

