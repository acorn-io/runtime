---
title: Args and Sensitive Data
---

## Args and settings

By design authors will build Acorns with defaults for the common usecase, but every deployment has some organization/environment specific configurations. Acorn allows consumers to customize behavior at runtime.

### Passing complex arguments

The easiest way to pass complex arguments to create a file in the local directory and pass with the `@` syntax.

config.yaml

```yaml
my:
  map:
    config: value
```

`acorn run registry.example.com/myorg/image --config @config.yaml`

## Binding secrets

To securely manage sensitive information while running Acorns the best practice is to use secrets. To accomplish this, the user needs to pre-create secrets before running the app.

### Show which secrets exist in the Acorn

To see which secrets will be created when the Acorn is deployed pass the `--help` flag on the Acorn image.

`acorn run registry.example.com/myorg/image --help`

There will be a `Secrets` line that lists the names of the secrets in the Acorn.

### Binding a secret at runtime

When running the Acorn you can bind in a secret with the `-s` option.

`acorn run -s my-predefined-creds:user-creds registry.example.com/myorg/image`

When this Acorn runs it will use the values in the `my-predefined-creds` secret.
