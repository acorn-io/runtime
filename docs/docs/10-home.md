---
title: Home
slug: /
---

:::caution

This site is very much a work in progress. The current structure will change drastically over time. For now, the most useful sections are the [Quick Start](#quickstart) and the [CLI Reference](100-Reference/01-command-line/acorn.md).
:::

## Overview

### Acorn

Acorn is a technology that brings the simplicity of running containers with Docker to Kubernetes. It does this by providing a familiar build, run, and deploy UX to Kubernetes. It provides a DSL to describe your application without the boilerplate of Kubernetes YAML files. With the application described in the Acorn DSL, it builds a portable artifact that contains everything the application needs to run, including the Docker images.

### What can I use Acorn for?

Acorn can be used to deploy containerized applications, including multi-container apps, onto any Kubernetes cluster, from developer laptops to production clusters in the cloud.

Packaging applications into a single portable artifact that includes all of the dependent OCI images and manifests. By having a single artifact to describe and run your application it makes it easier to move into air-gapped environments.

Running production workloads on the cluster.

Developing applications locally and moving to production without having to switch technology stacks.

## Quickstart

### Prerequisites

To try out Acorn you will need admin access to a Kubernetes cluster. Docker Desktop, Rancher Desktop, and K3s are all great options for trying out Acorn for testing/development.

### Install

On Linux and macOS you can use `brew` to quickly install Acorn.

For Windows and binary installs see the [installation docs](/Installation/installing#binary-install).

```shell
# Linux or macOS
brew install acorn-io/acorn/acorn

# verify binary (assume local directory)
acorn -v
```

For Windows and binary installs see the [installation docs](/Installation/installing#binary-install).

### Initialize Acorn on Kubernetes cluster

Before using Acorn on your cluster you need to initialize Acorn by running:

```shell
acorn install
```

You will only need to do this once per cluster.

### Build/Run First Acorn App

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

Next you can learn more about what you can do with Acorn in the [get started](/Get%20Started/Running%20an%20Acorn) guide.

## Next steps

* [Installation](/installation/installing)
* [Get Started](/Get%20Started/Running%20an%20Acorn)
* [Authoring Acornfiles](/Authoring%20Acornfiles/overview)
* [Running Acorn Apps in Production](/Running%20Acorn%20Apps%20in%20Production/args-and-secrets)
