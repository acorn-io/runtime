---
title: "acorn secret rm"
---
## acorn secret rm

Delete a secret

```
acorn secret rm [SECRET_NAME...] [flags]
```

### Examples

```

acorn secret rm my-secret
```

### Options

```
  -h, --help   help for rm
```

### Options inherited from parent commands

```
  -A, --all-projects        Use all known projects
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding current project
  -o, --output string       Output format (json, yaml, {{gotemplate}})
  -j, --project string      Project to work in
  -q, --quiet               Output only names
```

### SEE ALSO

* [acorn secret](acorn_secret.md)	 - Manage secrets

