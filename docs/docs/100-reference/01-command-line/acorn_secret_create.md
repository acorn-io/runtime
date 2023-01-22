---
title: "acorn secret create"
---
## acorn secret create

Create a secret

```
acorn secret create [flags] SECRET_NAME
```

### Examples

```

# Create secret with specific keys
acorn secret create --data key-name=value --data key-name2=value2 my-secret

# Read full secret from a file
acorn secret create --file secret.yaml my-secret

# Read key value from a file
acorn secret create --data @key-name=secret.yaml my-secret
```

### Options

```
      --data strings   Secret data format key=value or @key=filename to read from file
      --file string    File to read for entire secret in cue/yaml/json format
  -h, --help           help for create
      --type string    Secret type
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

