---
title: "acorn secret encrypt"
---
## acorn secret encrypt

Encrypt string information with clusters public key

```
acorn secret encrypt [flags] STRING
```

### Options

```
  -h, --help                 help for encrypt
      --plaintext-stdin      Take the plaintext from stdin
      --public-key strings   Pass one or more cluster publicKey values
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

