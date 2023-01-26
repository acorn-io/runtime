---
title: Args and Secrets
---

By design authors will build Acorns Images with defaults for the common use-case, but every deployment has some organization- or environment-specific configurations. Acorn allows consumers to customize behavior at runtime.

To learn which arguments are available for the Acorn image you can run:

```shell
acorn run [IMAGE] --help
```

### Passing simple arguments

To pass simple arguments, you pass the value after the argument name.

```shell
acorn run [IMAGE] --a-string "oneday" --int-arg 4 --bool-defaults-true --negate-a-true-bool=false
```

### Passing complex arguments

To pass complex arguments, create a file in the local directory and pass it to Acorn with the `@` syntax:

```yaml title="config.yaml"
my:
  map:
    config: value
```

`acorn run registry.example.com/myorg/image --config @config.yaml`

This is assuming that the Acorn defines a `config` arg where the contents should end up in.

## Binding secrets

To securely manage sensitive information while running Acorns the best practice is to use secrets. To accomplish this, the user needs to pre-create secrets before running the app.

### Discovering which secrets exist in the Acorn image

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

```shell
acorn run -s my-predefined-creds:user-creds registry.example.com/myorg/image
```

When this Acorn runs it will use the values in the `my-predefined-creds` secret.

## Encrypting data

### Overview

Encrypted secrets provide a way to pass sensitive information to Acorn apps through public channels. To accomplish this, Acorn uses a [libsodium sealed box](https://libsodium.gitbook.io/doc/public-key_cryptography/sealed_boxes) that is encrypted with the Acorn namespace's public key. This data can only be decrypted by Acorn in the intended namespace. For convenience data can be encrypted with multiple Acorn namespace public keys and put into the same data field. At runtime Acorn will try to decrypt the data with its own key pair. Once decrypted inside the namespace the values are stored in regular Kubernetes secrets for the app to consume. The primary use for encrypted secrets is to provide a mechanism to pass the data through untrusted systems like pipelines and command lines.

The [encryption reference section](../reference/encryption) explains how to use the Acorn public key to encrypt secrets in other languages.

### Encrypting

Acorn can encrypt plain text data as long as it is less than 4096 bytes. It is a good idea to keep the plain text value stored in another system like a password manager so that there is a backup of the sensitive data and that it can be re-encrypted if needed in a different Acorn namespace.

#### Encrypting for a single namespace

```shell
acorn secret encrypt "my secret data" 
# ACORNENC:eyIzclJrRH...
```

#### Encrypting for multiple targets

If you want to use the same sensitive data across multiple clusters or Acorn namespaces you can gather all of the public keys and encrypt all at once with the following command.

```shell
acorn secret encrypt --public-key <key1> [--public-key <keyN>...] "my secret data"
# ACORNENC:eyIzclJrRH...
```

The cipher text can be decrypted on all of the targets with that output.

### Using encrypted data

The encrypted text can be delivered to to the Acorn app by passing as an arg to the Acorn image (if one is predefined), or by placing the text into an existing secret that will be bound into the Acorn app at runtime.

When placing the data in either a secret or passing on the command line the entire string, including the `ACORNENC:` needs to be passed along.

If an argument was defined running the app would like:

```shell
acorn run db --root-password ACORNENC:eyIzclJrRH...
```

To create a secret that will be bound to the app at runtime.

```shell
acorn secret create --data key=ACORNENC:eyIzclJr... my-secret

acorn run -s my-secret:app-secret-name [IMAGE]
```

The secret will be decrypted when it is bound into the running app.

Alternatively, the Kubernetes secrets could be created using other tools and consumed by the running app by binding the secrets at runtime.

#### Complete encryption example

Using the following Acornfile we will pass in an encrypted secret using the methods above.

```acorn
args: password: ""

containers: app: {
  image: "alpine"
  env: {
    PASSWORD: "secret://user-password/pass"
  }
}

secrets: "user-password": {
  type: "opaque"
  data: password: "\(args.password)"
}
```

##### To pass in via an arg

```shell
acorn secret encrypt "secret password"
# ACORNENC:eyIzclJrRHBGRjlGamhUNHdHVGFJdnc4VTVNWDBwODBlb3NrOHl1NjFGT0FZIjoiZkU3RHB6TnF3ZkVacWRtaVBmdktKbGtTcTllSzdCa3VSM3ctT01YTG54a1RkZi1MR0Y5aWk2ZXhUMm9iWE02OC1Hc0RuQkJRWnZfUGNpQ0tzOVplIn0

acorn run . --password ACORNENC:eyIzclJrRHBGRjlGamhUNHdHVGFJdnc4VTVNWDBwODBlb3NrOHl1NjFGT0FZIjoiZkU3RHB6TnF3ZkVacWRtaVBmdktKbGtTcTllSzdCa3VSM3ctT01YTG54a1RkZi1MR0Y5aWk2ZXhUMm9iWE02OC1Hc0RuQkJRWnZfUGNpQ0tzOVplIn0
```

The environment variable will be set to "secret password" inside the running container.

Passing the password via the command line does open up the possibility of someone accidentally passing in plain text, but this is a quick and simple way to do it.

##### Pass via secret

WIth the same Acornfile as above, we will encrypt the data, create a secret and then bind in at runtime.

```shell
acorn secret encrypt "secret password"
# ACORNENC:eyIzclJrRHBGRjlGamhUNHdHVGFJdnc4VTVNWDBwODBlb3NrOHl1NjFGT0FZIjoiZkU3RHB6TnF3ZkVacWRtaVBmdktKbGtTcTllSzdCa3VSM3ctT01YTG54a1RkZi1MR0Y5aWk2ZXhUMm9iWE02OC1Hc0RuQkJRWnZfUGNpQ0tzOVplIn0

acorn secret create --data password=ACORNENC:eyIzclJrRHBGRjlGamhUNHdHVGFJdnc4VTVNWDBwODBlb3NrOHl1NjFGT0FZIjoiZkU3RHB6TnF3ZkVacWRtaVBmdktKbGtTcTllSzdCa3VSM3ctT01YTG54a1RkZi1MR0Y5aWk2ZXhUMm9iWE02OC1Hc0RuQkJRWnZfUGNpQ0tzOVplIn0
# pre-created-secret

acorn run -s pre-created-secret:user-password .
# wild-horse
```

Instead of manually creating the secret via command line in the second step, a separate process could apply the manifests with the encrypted value.
