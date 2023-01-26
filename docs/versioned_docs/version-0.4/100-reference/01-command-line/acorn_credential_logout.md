---
title: "acorn credential logout"
---
## acorn credential logout

Remove registry credentials

```
acorn credential logout [flags] [SERVER_ADDRESS]
```

### Examples

```

acorn logout ghcr.io
```

### Options

```
  -h, --help   help for logout
```

### Options inherited from parent commands

```
  -A, --all-namespaces      Namespace to work in
      --context string      Context to use in the kubeconfig file
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Location of a kubeconfig file
      --namespace string    Namespace to work in (default "acorn")
  -o, --output string       Output format (json, yaml, {{gotemplate}})
  -q, --quiet               Output only names
```

### SEE ALSO

* [acorn credential](acorn_credential.md)	 - Manage registry credentials

