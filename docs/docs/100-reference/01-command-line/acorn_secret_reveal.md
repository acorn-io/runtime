---
title: "acorn secret reveal"
---
## acorn secret reveal

Reveal the values of a secret.

```
acorn secret reveal [flags] [SECRET_NAME...]
```

### Examples

```
acorn secret reveal foo-secret-ab123
```

### Options

```
  -h, --help            help for reveal
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

* [acorn secret](acorn_secret.md)	 - Manage secrets

