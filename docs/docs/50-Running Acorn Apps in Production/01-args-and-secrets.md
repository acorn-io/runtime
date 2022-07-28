---
title: Args and Secrets
---

## Args and settings

By design authors will build Acorns Images with defaults for the common usecase, but every deployment has some organization/environment specific configurations. Acorn allows consumers to customize behavior at runtime.

To learn which arguments are available for the Acorn Image you can run:

```shell
acorn run [IMAGE] --help
```

### Passing simple arguments

To pass simple arguments, you pass the value after the argument name.

```shell
acorn run [IMAGE] --a-string "oneday" --int-arg 4 --bool-defaults-true --negate-a-true-bool=false
```

### Passing complex arguments

To pass complex arguments is to create a file in the local directory and pass it to Acorn with the `@` syntax:

```yaml
# config.yaml
my:
  map:
    config: value
```

`acorn run registry.example.com/myorg/image --config @config.yaml`

This is assuming that the Acorn defines a `config` arg where the contents should end up in.

## Binding secrets

To securely manage sensitive information while running Acorns the best practice is to use secrets. To accomplish this, the user needs to pre-create secrets before running the app.

### Discovering which secrets exist in the Acorn Image

To see which secrets will be created when the Acorn is deployed pass the `--help` flag on the Acorn image.

```shell
acorn run registry.example.com/myorg/image --help
```

There will be a `Secrets` line that lists the names of the secrets in the Acorn.

### Creating a secret

To create a secret you can use the `acorn secret create` command

```shell
# Create the secret
> acorn secret create --data username=user0 --data password=supersecret1 my-app-secret-creds
my-app-secret-creds
```

In the above example the values from `my-app-secret-creds` will now be available to bind in the secret.

### Binding a secret at runtime

When running the Acorn you can bind in a secret with the `-s` option.

`acorn run -s my-predefined-creds:user-creds registry.example.com/myorg/image`

When this Acorn runs it will use the values in the `my-predefined-creds` secret.
