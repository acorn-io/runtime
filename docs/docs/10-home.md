---
title: Home
slug: /
---

:::caution

Acorn is a work in progress.  Features will evolve over time and there may be breaking changes between releases.  Please give us your feedback in [Slack](https://slack.acorn.io), [Discussions](https://github.com/acorn-io/acorn/discussions), or [Issues](https://github.com/acorn-io/acorn/issues)!
:::

## Overview

### What is Acorn?

Acorn is an application packaging and deployment framework that simplifies running apps on Kubernetes. Acorn is able to package up all of an applications Docker images, configuration, and deployment specifications into a single Acorn image artifact. This artifact is publishable to any OCI container registry allowing it to be deployed on any dev, test, or production Kubernetes.  The portability of Acorn images enables developers to develop applications locally and move to production without having to switch tools or technology stacks.

Developers create Acorn images by describing the application configuration in an [Acornfile](/authoring/overview). The Acornfile describes the whole application without all of the boilerplate of Kubernetes YAML files. The Acorn CLI is used to build, deploy, and operate Acorn images on any Kubernetes cluster.

## Quickstart

### Prerequisites

To try out Acorn you will need admin access to a Kubernetes cluster. Docker Desktop, Rancher Desktop, and K3s are all great options for trying out Acorn for testing/development.

### Install

On Linux and macOS you can use `brew` to quickly install Acorn.

For Windows and binary installs see the [installation docs](/installation/installing#binary-install).

```shell
# Linux or macOS
brew install acorn-io/cli/acorn

# verify binary (assume local directory)
acorn -v
```

For Windows and binary installs see the [installation docs](/installation/installing#binary-install).

### Initialize Acorn on Kubernetes cluster

Before using Acorn on your cluster you need to initialize Acorn by running:

```shell
acorn install
```

You will only need to do this once per cluster.

### Build/Run First Acorn app

Create a new `Acornfile` in your working directory and add the following contents.

```cue
containers: {
 web: {
  image: "nginx"
  ports: publish: "80/http"
  files: {
   // Simple index.html file
   "/usr/share/nginx/html/index.html": "<h1>My First Acorn!</h1>"
  }
 }
}
```

Save the file. What this file defines is a container called `web` based on the nginx container image from DockerHub. It also declares that port 80 should be exposed and that it will expose an http protocol service. We are also customizing the `index.html` file as part of our packaging process. The contents of the file will be added during the build process.

Now you will need to build your Acorn from this file by typing `acorn build .`. This will launch an Acorn builder and development registry into your Kubernetes cluster and build the Acorn image.

```shell
acorn run .
#[+] Building 0.8s (5/5) FINISHED
# => [internal] load .dockerignore
# => => transferring context: 2B  
# ...
#small-butterfly

```

Our Acorn has started and is named `small-butterfly`.

To check the status of our app we can run the following.

```shell
acorn apps small-butterfly
#NAME         IMAGE                                                              HEALTHY   UPTODATE   CREATED              ENDPOINTS                                           MESSAGE
#little-dew   60d803258f7aa2680e4910c526485488949835728a2bc3519c09f1b6b3be1bb3   1         1          About a minute ago   http://web.little-dew.local.on-acorn.io => web:80   OK
```

In Chrome or Firefox browsers you can now open the URL listed under the endpoints column to see our app.

Next you can learn more about what you can do with Acorn in the [getting started](/getting-started) guide.

## Next steps

* [Installation](/installation/installing)
* [Getting Started](/getting-started)
* [Authoring Acornfiles](/authoring/overview)
* [Running Acorn apps in Production](/production/args-and-secrets)
