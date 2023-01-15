---
title: "acorn secret reveal"
---
## acorn secret reveal

Manage secrets

```
acorn secret reveal [flags] [SECRET_NAME...]
```

### Examples

```

acorn secret
```

### Options

```
  -h, --help            help for reveal
  -o, --output string   Output format (json, yaml, {{gotemplate}})
  -q, --quiet           Output only names
```

### Options inherited from parent commands

```
  -A, --all-projects        Use all known projects
      --context string      Context to use in the resolved kubeconfig file
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding current project
      --namespace string    Namespace to work in resolved connection (default "acorn")
  -j, --project string      Project to work in
```

### SEE ALSO

* [acorn secret](acorn_secret.md)	 - Manage secrets

