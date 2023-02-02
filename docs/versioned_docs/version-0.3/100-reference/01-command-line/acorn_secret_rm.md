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
  -A, --all-namespaces      Namespace to work in
      --context string      Context to use in the kubeconfig file
      --kubeconfig string   Location of a kubeconfig file
      --namespace string    Namespace to work in (default "acorn")
  -o, --output string       Output format (json, yaml, {{gotemplate}})
  -q, --quiet               Output only names
```

### SEE ALSO

* [acorn secret](acorn_secret.md)	 - Manage secrets

