---
title: Credentials
---

Credentials refer to credentials used to pull from and/or push to OCI 
registries. In the future credentials in Acorn may be used for different
types of credential, but as it stand they are only used for OCI registries.

## Storage

Credentials are store within the cluster in a namespaced secret. Acorn
API does not give access to the secret values of the credential, namely
the password or token. If a user has access to use the credential that
does not mean they can see the credential value. This makes it safe
to share credentials in a team setting.

## Scope/Access

Credentials are valid for all apps and images in a namespace. Any use
that has privileges to push or pull and image will implicitly be using
the credentials stored in that namespace. Similarily any app that is
deploy will use the credentials available in the namespace to pull the
Acorn image and referenced Docker images.

## CLI

Credentials are managed with the `acorn credential` command.
