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
  -l, --local-storage     Store credential on local client for push, pull, and build (not run)
  -p, --password string   Password
      --password-stdin    Take the password from stdin
      --skip-checks       Bypass login validation checks
  -u, --username string   Username
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

