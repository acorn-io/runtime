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
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding current project
  -j, --project string      Project to work in
```

### SEE ALSO

* [acorn](acorn.md)	 - 

