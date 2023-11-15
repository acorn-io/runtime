---
title: "acorn secret update"
---
## acorn secret update

Update a secret

```
acorn secret update [flags] SECRET_NAME
```

### Examples

```

# Create secret with specific keys
acorn secret update --data key-name=value --data key-name2=value2 my-secret

# Read full secret from a file. The file should have a type and data field.
acorn secret update --file secret.yaml my-secret

# Read key value from a file
acorn secret update --data @key-name=secret.yaml my-secret
```

### Options

```
      --data strings   Secret data format key=value or @key=filename to read from file
      --file string    File to read for entire secret in aml/yaml/json format
  -h, --help           help for update
      --type string    Secret type
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

