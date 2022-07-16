---
title: Publishing Acorn Images
---

## Preparing to run apps in production

Once the application is in a state where it needs to be moved to dev, test, and production you will need to build and publish an Acorn image to a registry.

## Build and tagging an image

To publish your image run the familiar build command with a `-t` option to tag the image. A tag will include the domain name of the registry and the URI path for the organization. Commonly, the URI path is a variation of `/<organization>/<app-name>:<version>`

An example would be:

`acorn build -t ghcr.io/acorn-io/acorn:v1.0 .`

This is very similar to the Docker container workflow:

`docker build -t registry.example.com/myorg/image:version .`

You can use the tag to reference the built Acorn image to run, push, and update image.
