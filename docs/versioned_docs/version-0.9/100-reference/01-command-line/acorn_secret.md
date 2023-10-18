---
title: "acorn secret"
---
## acorn secret

Manage secrets

```
acorn secret [flags] [SECRET_NAME...]
```

### Examples

```

acorn secret
```

### Options

```
  -h, --help            help for secret
  -o, --output string   Output format (json, yaml, {{gotemplate}})
  -q, --quiet           Output only names
```

### Options inherited from parent commands

```
      --config-file string   Path of the acorn config file to use
      --debug                Enable debug logging
      --debug-level int      Debug log level (valid 0-9) (default 7)
      --kubeconfig string    Explicitly use kubeconfig file, overriding the default context
  -j, --project string       Project to work in
```

### SEE ALSO

* [acorn](acorn.md)	 - 
* [acorn secret create](acorn_secret_create.md)	 - Create a secret
* [acorn secret encrypt](acorn_secret_encrypt.md)	 - Encrypt string information with clusters public key
* [acorn secret reveal](acorn_secret_reveal.md)	 - Manage secrets
* [acorn secret rm](acorn_secret_rm.md)	 - Delete a secret

