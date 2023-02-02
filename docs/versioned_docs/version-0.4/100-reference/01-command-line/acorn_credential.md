---
title: "acorn credential"
---
## acorn credential

Manage registry credentials

```
acorn credential [flags] [SERVER_ADDRESS...]
```

### Examples

```

acorn credential
```

### Options

```
  -h, --help            help for credential
  -o, --output string   Output format (json, yaml, {{gotemplate}})
  -q, --quiet           Output only names
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
* [acorn credential login](acorn_credential_login.md)	 - Add registry credentials
* [acorn credential logout](acorn_credential_logout.md)	 - Remove registry credentials

