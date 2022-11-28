---
title: "acorn login"
---
## acorn login

Add registry credentials

```
acorn login [flags] [SERVER_ADDRESS]
```

### Examples

```

acorn login ghcr.io
```

### Options

```
  -h, --help              help for login
  -p, --password string   Password
      --password-stdin    Take the password from stdin
      --skip-checks       Bypass login validation checks
  -u, --username string   Username
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

