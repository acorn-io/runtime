---
title: Running an Acorn
---
## Overview

In this guide you will walk through:

* Running an existing Acorn
* Building and running an Acorn for your project

## Running an Acorn App

If you would like to run an Acorn from a registry the command is:

`acorn run registry.example.com/myorg/app-image`

There is no need to pull the image ahead of time, Acorn will pull it if the image is not on the host.

To see what arguments are available to customize the Acorn add `--help` after the image name.

`acorn run registry.example.com/myorg/app-image --help`

To pass values:

`acorn run registry.example.com/myorg/app-image --a-false-bool=false --replicas 2`
