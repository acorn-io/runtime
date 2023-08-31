---
title: Encryption
---


## Public keys

To get the public key for a namespace run the following.

```shell
acorn info
#---
#client:
#  version:
# ...
#namespace:
#  publicKeys:
#  - keyID: 3rRkDpFF9FjhT4wGTaIvw8U5MX0p80eosk8yu61FOAY
#server:
#  apiServerImage: ghcr.io/acorn-io/runtime:main
#  config:
#....
```

The public key is available under the namespace top level key. The value is encoding using golang's base64.RawURLEncoding encoder. This means it doesn't have padding and is safe to use in URLs. In some languages, like Python, you might need to re-add the padding before being able to use the key.

Using the [pynacl](https://pynacl.readthedocs.io/en/latest/public/#nacl-public-sealedbox) package for instance the key needs to have the padding added back in.

```python
import base64
from nacl.public import SealedBox, PublicKey
from nacl.encoding import URLSafeBase64Encoder
...
key = "3rRkDpFF9FjhT4wGTaIvw8U5MX0p80eosk8yu61FOAY"
padded_key = key + '=' * (-len(key) % 4)

pkAcorn = PublicKey(padded_key, encoder=URLSafeBase64Encoder)
...
```

## Encrypting the plain text

The plain text is encrypted using a libsodium sealed secret. These make use of an ephemeral public and private key pair from the sender and the public key of the receiver. In Python using the pynacl library:

```python
import base64
from nacl.public import SealedBox, PublicKey
from nacl.encoding import URLSafeBase64Encoder

...
sealed_box = SealedBox(pkAcorn)
ciphertext = sealed_box.encrypt(message)
...
 ```

## Acorn message format

Acorn expects the secret to be in a string in the form of:

```shell
ACORNENC:base64.RawURLEncode.EncodeToString("{"publicKey":"base64.RawURLEncoding.EncodeToString(ciphertext)",...})
```

Where the `publicKey` is the string value returned from `acorn info`. The object can include multiple `publicKey:ciphertext` items. Acorn will attempt to decrypt the values with the key available in the namespace.

## Private keys

Acorn creates a public/private key pair that is tied to the underlying Acorn namespace and UID. If you delete the namespace or uninstall Acorn the encrypted data can not be unencrypted. The private key is stored outside of the users Acorn namespace to prevent accidental exposure of the key.

## Additional info

- [libsodium](https://doc.libsodium.org/bindings_for_other_languages/)
