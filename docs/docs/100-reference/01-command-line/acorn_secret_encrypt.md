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

