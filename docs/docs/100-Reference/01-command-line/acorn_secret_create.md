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
acorn secret create --data key=value --data key2=value2 my-secret,

# Read full secret from a file
acorn secret create --file secret.yaml my-secret

# Read key value from a file
acorn secret create --data @key=secret.yaml my-secret
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
  -A, --all-namespaces      Namespace to work in
      --context string      Context to use in the kubeconfig file
      --kubeconfig string   Location of a kubeconfig file
      --namespace string    Namespace to work in (default "acorn")
  -o, --output string       Output format (json, yaml, {{gotemplate}})
  -q, --quiet               Output only names
```

### SEE ALSO

* [acorn secret](acorn_secret.md)	 - Manage secrets

