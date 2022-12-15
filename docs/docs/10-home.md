---
title: Home
slug: /
---

:::caution

Acorn is a work in progress.  Features will evolve over time and there may be breaking changes between releases.  Please give us your feedback in [Slack](https://slack.acorn.io), [Discussions](https://github.com/acorn-io/acorn/discussions), or [Issues](https://github.com/acorn-io/acorn/issues)!
:::

## Overview

### What is Acorn?

Acorn is an application packaging and deployment framework that simplifies running apps on Kubernetes. Acorn is able to package up all of an application's Docker images, configuration, and deployment specifications into a single Acorn image artifact. This artifact is publishable to any OCI container registry allowing it to be deployed on any dev, test, or production Kubernetes.  The portability of Acorn images enables developers to develop applications locally and move to production without having to switch tools or technology stacks.

Developers create Acorn images by describing the application configuration in an [Acornfile](/authoring/overview). The Acornfile describes the whole application without all of the boilerplate of Kubernetes YAML files. The Acorn CLI is used to build, deploy, and operate Acorn images on any Kubernetes cluster.

### Acorn Workflow

The following figure illustrates the steps a user takes when using Acorn.

1. The user authors an Acornfile to describe the application.
2. The user invokes the Acorn CLI to build an Acorn image from the Acornfile.
3. The Acorn image is pushed to an OCI registry.
4. The user invokes the Acorn CLI to deploy the Acorn image onto an Acorn runtime, which can be installed on any Kubernetes cluster.

![Acorn Workflow](/img/acorn.workflow.png)

### Acorn vs. Helm

Helm is a popular package manager for Kubernetes. After working with Helm charts for many years, we built Acorn
specifically to offer a simplified application deployment experience for Kubernetes. Here are some of the
differences between Acorn and Helm.

1. Helm charts are templates for Kubernetes YAML files, whereas Acornfiles define application-level constructs. Acorn is
a layer of abstraction on top of Kubernetes. Acorn users do not work with Kubernetes YAML files directly. By design, no Kubernetes
knowledge is needed to use Acorn. 

2. Helm users can package any Kubernetes workload into Helm charts, whereas Acorn is designed to package applications and not
system-level drivers, plugins, and agents. Acorn supports any type of application, stateless and stateful. Applications
run in their own namespaces. Applications do not need privileged containers. Applications run on Kubernetes but do not call the
underlying Kubernetes API or use the underlying etcd as a database by defining custom resources.

3. Acornfiles define application-level constructs such as Docker containers, application configuration, and application
deployment specifications. Acorn brings structure to application deployment on Kubernetes. This is in marked contrast with
unconstrained use of Kubernetes YAML files in Helm charts.

We hope Acorn will simplify packaging and deployment of applications on Kubernetes. 



## Quickstart

### Prerequisites

To try out Acorn you will need admin access to a Kubernetes cluster. Docker Desktop, Rancher Desktop, and K3s are all great options for trying out Acorn for testing/development.

### Install

On Linux and macOS you can use `brew` to quickly install Acorn.

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

:::note
Acorn has a handful of installation requirements. If you're installing to your desktop Kubernetes cluster, you likely meet the requirements. If you're installing into a remote cluster, please review the detailed [installation instructions](/installation/installing).
:::

You will only need to do this once per cluster.

### Build/Run First Acorn app

Create a new `Acornfile` in your working directory and add the following contents.

```acorn
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
#green-bush

```

Our Acorn has started and is named `green-bush`.

To check the status of our app we can run the following.

```shell
acorn apps green-bush
#NAME         IMAGE                                                              HEALTHY   UPTODATE   CREATED              ENDPOINTS                                                              MESSAGE
#green-bush   60d803258f7aa2680e4910c526485488949835728a2bc3519c09f1b6b3be1bb3   1         1          About a minute ago   http://web-nginx-green-bush-6cc6aeba547e.local.on-acorn.io => web:80   OK
```

In Chrome or Firefox browsers you can now open the URL listed under the endpoints column to see our app.

Next you can learn more about what you can do with Acorn in the [getting started](/getting-started) guide.

## Next steps

* [Installation](/installation/installing)
* [Getting Started](/getting-started)
* [Authoring Acornfiles](/authoring/overview)
* [Running Acorn Apps](/running/args-and-secrets)
