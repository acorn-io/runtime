---
title: "acorn secret edit"
---
## acorn secret edit

Edits a secret

```
acorn secret edit SECRET_NAME [flags]
```

### Examples

```
acorn secret edit my-secret
```

### Options

```
  -h, --help   help for edit
```

### Options inherited from parent commands

```
      --config-file string   Path of the acorn config file to use
      --debug                Enable debug logging
      --debug-level int      Debug log level (valid 0-9) (default 7)
      --kubeconfig string    Explicitly use kubeconfig file, overriding the default context
  -o, --output string        Output format (json, yaml, {{gotemplate}})
  -j, --project string       Project to work in
  -q, --quiet                Output only names
```

### SEE ALSO

* [acorn secret](acorn_secret.md)	 - Manage secrets

