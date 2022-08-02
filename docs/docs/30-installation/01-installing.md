---
title: Installing
---


To install Acorn, you will need the Acorn CLI and a Kubernetes cluster. Follow one of the methods below to install the Acorn CLI and then install onto the [Kubernetes cluster](#installing-acorn-onto-kubernetes-clusters).

```shell
acorn install
```

## Acorn CLI

### macOS and Linux

The preferred method for installing on Mac and Linux is to use the brew package manager.

You can install the latest Acorn CLI with the following:

```shell
brew install acorn-io/cli/acorn
```

You can also follow the binary installation below.

### Binary install

You can download the Acorn CLI binary from the project's [GitHub page](https://github.com/acorn-io/acorn/releases).

Download the correct binary for your platform.

#### macOS Binary install

Download either the universal DMG or the tar.gz file.

For the DMG run through the installer.

For the tar.gz download:

```shell
tar -zxvf ~/Downloads/acorn-v<version>-macos-universal.tar.gz
cp ~/Downloads/acorn /usr/local/bin/acorn
```

 *Note: if using zsh you will need to make sure ulimit -f can handle files > 140MB*

#### Linux

Download the tar.gz binary for your architecture. Uncompress and move the binary to your PATH.

```shell
 tar -zxvf ~/Downloads/acorn-v<version>-linux-<arch>.tar.gz
 mv ~/Downloads/acorn /usr/local/bin
```

### Windows

For Windows systems, please follow the binary install method.

## Installing Acorn onto Kubernetes clusters

Acorn will need to be initialized on each Kubernetes cluster you plan to use it on.

```shell
acorn install
```

Acorn can install onto any type of Kubernetes cluster capable of running normal workloads. The following are requirements and considerations for installing Acorn.

### Privileges
You must have cluster admin privileges to install Acorn. See our [RBAC documentation](architecture/security-considerations#rbac) for more details.

### Ingress
Acorn can expose your applications as publicly accessible URLS. For this to work, your Kubernetes cluster must have an [ingress controller](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/).

### Storage
Acorn supports persistent storage through the use of volumes. For this to work, your Kubernetes cluster must have a [default storage class](https://kubernetes.io/docs/concepts/storage/storage-classes/).

### Local Development Clusters
For local development, Acorn has been tested with Rancher Desktop, Docker Desktop, and Minikube. If you are using one of these systems, please consider the following:

**Rancher Desktop** comes with a working ingress and storage class by default. No additional configuration is necessary.

**Docker Desktop** comes with a storage class, but not an ingress. If you're using Rancher Desktop and don't have one installed, Acorn will install the [NGINX Ingress Controller](https://kubernetes.github.io/ingress-nginx/) for you.

**Minikube** comes with a storage class, but requires that you [enable ingress explicitly](https://kubernetes.io/docs/tasks/access-application-cluster/ingress-minikube/#enable-the-ingress-controller) with the following command:
```shell
minikube addons enable ingress
```
It's not obvious in the above minikube documentation, but after enabling ingress, if you want to access your applications locally, you must also run:
```shell
minikube tunnel
```
