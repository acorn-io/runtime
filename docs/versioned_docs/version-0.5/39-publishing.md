---
title: Publishing Acorn Images
---

Once the application is in a state where it is ready to move to test and production you will need to build an Acorn image and publish it to a registry. Acorn images are only accessible to the Acorn namespace they were built in. In order to use them in other namespaces or on other Kubernetes clusters, the images need to be tagged and published to a registry.

## Building and tagging an Acorn image

To publish your image run the familiar build command with a `-t` option to tag the image. A tag will include the FQDN of the registry and the URI path for the image. Commonly, the URI path is a variation of `/<organization>/<app-name>:<version>`

An example would be:

```shell
acorn build -t ghcr.io/acorn-io/acorn:v1.0 .
```

This is very similar to the Docker build workflow:

```shell
docker build -t index.docker.io/<org>/<image>:<version> .
```

You can use the tag to reference the built Acorn image to run, push, and update it.

## Tagging existing Acorn images

If you want to push a local Acorn image to another registry, or move from a SHA to a friendly name, you can tag the image. The command is:

```shell
acorn tag <current_image> <new_image>
```

To tag and prepare to push to Dockerhub `index.docker.io/my-org/image:version` run:

```shell
acorn tag [MY-IMAGE] /myorg/image:latest
```

## Pushing Acorn image to production

Once the image is tagged, it is ready to be pushed to the remote registry.

### Logging in

First you will need to login with credentials that have push access to the remote registry.

```shell

# Docker hub
acorn login index.docker.io

# - or -

#GitHub container registry
acorn login ghcr.io
```

You will be prompted for your username and password to login. If your company has an internal registry you can login subtituting `ghcr.io` for your organizations registry domain.

### Push the image

Pushing to a registry requires 2 things:

1. User is logged in and authorized.
1. Image is tagged for the remote registry.

```shell
acorn push index.docker.io/myorg/image:v1.0
```

## Pulling / Running the Acorn image

Once the image has been published to a registry, it can be run on other clusters that have access to that registry. You can run the acorn and the Acorn image will automatically be pulled.

```shell
acorn run index.docker.io/myorg/image:v1.0
```

You can manually pull the Acorn image:

```shell
acorn pull index.docker.io/myorg/image:v1.0
```

## Additional Information

* See [Credentials](./architecture/security-considerations) docs for details on how registry credentials are scoped and stored.
