---
title: "acorn credential login"
---
## acorn credential login

Add registry credentials

```
acorn credential login [flags] [SERVER_ADDRESS]
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
      --context string      Context to use in the resolved kubeconfig file
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding current project
      --namespace string    Namespace to work in resolved connection (default "acorn")
  -o, --output string       Output format (json, yaml, {{gotemplate}})
  -j, --project string      Project to work in
  -q, --quiet               Output only names
```

### SEE ALSO

* [acorn credential](acorn_credential.md)	 - Manage registry credentials

