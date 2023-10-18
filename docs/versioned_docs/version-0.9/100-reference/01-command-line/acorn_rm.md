---
title: "acorn rm"
---
## acorn rm

Delete an acorn, optionally with it's associated secrets and volumes

```
acorn rm [flags] ACORN_NAME [ACORN_NAME...]
```

### Examples

```

acorn rm ACORN_NAME
acorn rm --volumes --secrets ACORN_NAME
```

### Options

```
  -a, --all              Delete all associated resources (volumes, secrets)
  -f, --force            Do not prompt for delete
  -h, --help             help for rm
      --ignore-cleanup   Delete acorns without running delete jobs
  -s, --secrets          Delete acorn and associated secrets
  -v, --volumes          Delete acorn and associated volumes
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

