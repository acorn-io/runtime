---
title: Managing Images
---

## Listing images

To see what images are available locally in your cluster.

`acorn images`

## Tagging images

If you want to tag a local image to push to another registry, or move from a SHA to a friendly name, you can tag the image. The command is:

`acorn tag <current_image> <new_image>`

To tag and prepare to push to registry.example.com/myorg/image:version run:

`acorn tag myimage registry.example.com/myorg/image:latest`

## Pull from public registry

To pull a remote Acorn image locally you can use the `pull` subcommand.

`acorn pull registry.example.com/myorg/image:v1.0`

## Login to a private registry

Registries are typically protected by an authentication/authorization system to prevent unwanted pushing/pulling of artifacts. To login to a registry users can do:

`acorn login registry.example.com`

or

`acorn credentials login registry.example.com`

The user will be prompted for the username and password for the registry.

Optionally, the user may use the `--username <USER>` flag and the `--password <PASS>` flags on the login subcommand.

Also available is the `--password-stdin` flag that takes the password from stdin.

### Pull from private registry

Pulling from a private registry requires that you are authorized by having logged in. Otherwise it is the same as doing a regular pull.

## Push to a registry

Pushing to a registry requires

1. User is logged in and authorized.
1. Image is tagged to the remote server.

```shell
> acorn credentials login --username USER remote-registry.example.com
> acorn tag local-image remote-registry.example.com/myorg/image:v1.0
> acorn push remote-registry.example.com/myorg/image:v1.0
```

## Deleting images

### Specific image

List all of the images, and remove the image based on the sha.

```shell
> acorn images
REPOSITORY   TAG       IMAGE-ID
<none>       <none>    4e3180335940
<none>       <none>    a34173b24c2a
<none>       <none>    5dc869d262b3
```

Then remove the image by IMAGE-ID

`acorn rm -i 4e3180335940`

The `-i` option specifies the image.

### All images

Warning: This removes ALL images on current clusters namespace. If you want to delete all images on your cluster you can use:

`acorn rm -i $(acorn images -a -q)`
