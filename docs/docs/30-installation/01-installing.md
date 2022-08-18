---
title: Installing
---


To install Acorn, you will need the Acorn CLI and a Kubernetes cluster. Follow one of the methods below to install the Acorn CLI and then install onto the [Kubernetes cluster](#installing-acorn-onto-kubernetes-clusters).

```shell
acorn install
```

## Acorn CLI

### Homebrew (macOS & Linux)

The preferred method for installing on Mac and Linux is to use the brew package manager.

You can install the latest Acorn CLI with the following:

```shell
brew install acorn-io/cli/acorn
```

You can also follow the binary installation below.

### curl|sh install (macOS & Linux)

If you don't have homebrew, you can install the CLI with this one-liner:

```shell
curl https://get.acorn.io | sh
```

### Manual install

You can download the Acorn CLI binary from the project's [GitHub page](https://github.com/acorn-io/acorn/releases).

Download the correct binary for your platform.

#### macOS

Download either the universal DMG or the tar.gz file.

For the DMG run through the installer.

For the tar.gz download:

```shell
tar -zxvf ~/Downloads/acorn-v<version>-macos-universal.tar.gz
cp ~/Downloads/acorn /usr/local/bin/acorn
```

 *Note: if using zsh you will need to make sure ulimit -f can handle files > 140MB*

#### Linux

Download the tar.gz archive for your architecture. Uncompress and move the binary to your PATH.

```shell
 tar -zxvf ~/Downloads/acorn-v<version>-linux-<arch>.tar.gz
 mv ~/Downloads/acorn /usr/local/bin
```

#### Windows

Uncompress and move the binary to your PATH.

#### Development Binaries (main build)

The last successful build from the HEAD of the main branch is available for
[macOS](https://cdn.acrn.io/cli/default_darwin_amd64_v1/acorn),
[Linux](https://cdn.acrn.io/cli/default_linux_amd64_v1/acorn), and
[Windows](https://cdn.acrn.io/cli/default_windows_amd64_v1/acorn.exe)

### Shell completion

For best developer experience, shell autocompletion is provided, but the acorn cli subcommand is hidden.
To set autocompletion for the current terminal session, use the command that matches your shell:
```
source <(acorn completion bash)
source <(acorn completion zsh)
acorn completion fish | source
```

For permanent effect add the same line to your shell specific profile:
- ~/.bashrc
- ~/.zshrc
- ~/.config/fish/config.fish

## Installing Acorn onto Kubernetes clusters

Acorn will need to be initialized on each Kubernetes cluster you plan to use it on.

```shell
acorn install
```

Acorn can install onto any type of Kubernetes cluster capable of running normal workloads. The following are requirements and considerations for installing Acorn.

### Privileges

You must have cluster admin privileges to install Acorn. See our [RBAC documentation](/architecture/security-considerations#rbac) for more details.

### Ingress and Service LoadBalancers

Acorn can publish your applications as publicly accessible endpoints.

For this to work, your Kubernetes cluster must have an [ingress controller](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/) for HTTP endpoints and means for fulfilling [services of type LoadBalancer](https://kubernetes.io/docs/concepts/services-networking/service/#loadbalancer) for non-HTTP endpoints, such as TCP endpoints.

### Storage

Acorn supports persistent storage through the use of volumes. For this to work, your Kubernetes cluster must have a [default storage class](https://kubernetes.io/docs/concepts/storage/storage-classes/).

### Local Development Clusters

For local development, Acorn has been tested with Rancher Desktop, Docker Desktop, and Minikube. If you are using one of these systems, please consider the following:

**Rancher Desktop** comes with a working ingress controller, service loadbalancer solution, and storage class by default. No additional configuration is necessary.

**Docker Desktop** comes with a storage class and service loadbalancer solution, but not an ingress. If you're using Docker Desktop and don't have one installed, Acorn will install the [NGINX Ingress Controller](https://kubernetes.github.io/ingress-nginx/) for you.

**Minikube** comes with a default storage class, but requires that you [enable ingress explicitly](https://kubernetes.io/docs/tasks/access-application-cluster/ingress-minikube/#enable-the-ingress-controller) with the following command:

```shell
minikube addons enable ingress
```

It's not obvious in the above minikube documentation, but after enabling ingress, if you want to access your applications locally, you must also run:

```shell
minikube tunnel
```

The tunnel command also services as the service loadbalancer solution. If it is running, your TCP services will be published to `localhost`.
