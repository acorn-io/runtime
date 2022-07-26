---
title: Running an Acorn
---
## Overview

In this guide you will walk through:

* Running an existing Acorn
* Building and running an Acorn for your project

## Running an Acorn app

If you would like to run an acorn from a registry the command is:

`acorn run registry.example.com/myorg/app-image`

There is no need to pull the image ahead of time, Acorn will pull it if the image is not on the host.

To see what arguments are available to customize the Acorn add `--help` after the image name.

`acorn run registry.example.com/myorg/app-image --help`

To pass values:

`acorn run registry.example.com/myorg/app-image --a-false-bool=false --replicas 2`

## Starting the Acorn Dashboard

In addition to the CLI, we want to call out the Acorn Dashboard as another way to quickly view Acorn applications on your cluster. From the dashboard, you can gain access to the containers logs, quickly find URL links to your apps, and manage volumes and secrets.

To start the dashboard, run the following command:
`acorn dashboard`
If you run the command now, you will see your application running a link to access any of its published ports.

![Dashboard](/img/dashboard.png)
