---
title: Installing
---


To install Acorn, you will need the Acorn CLI and a Kubernetes cluster. Follow one of the methods below to install the Acorn CLI and then install onto the [Kubernetes cluster](#installing-acorn-onto-kubernetes-clusters).

```shell
acorn install
```

In many cases, the default installation options for Acorn are sufficient, but there are a number of options you can use to customize Acorn. See our [Installation Options](./options) page for more details.

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

### Scoop (Windows)

You can install the latest Acorn CLI with the following:

```shell
scoop install acorn
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

The Acorn CLI supports command autocompletion. If you installed acorn using homebrew, this is already configured for you. If you installed using the manual or curl|sh method, you must enable shell completion yourself.

To set autocompletion for the current terminal session, use the command that matches your shell:

```shell
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

### Cluster Requirements
#### Kubernetes version

Acorn requires Kubernetes 1.23 or greater.

#### Privileges

You must have cluster admin privileges to install Acorn. See our [RBAC documentation](../architecture/security-considerations#rbac) for more details.

#### Ingress and Service LoadBalancers

Acorn can publish your applications as publicly accessible endpoints.

For this to work, your Kubernetes cluster must have an [ingress controller](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/) for HTTP endpoints and means for fulfilling [services of type LoadBalancer](https://kubernetes.io/docs/concepts/services-networking/service/#loadbalancer) for non-HTTP endpoints, such as TCP endpoints.

#### Storage

Acorn supports persistent storage through the use of volumes. For this to work, your Kubernetes cluster must have a [default storage class](https://kubernetes.io/docs/concepts/storage/storage-classes/).

### YAML Based Install

If you would like to see the generated objects prior to installing to your Kubernetes cluster run: 

```shell
acorn install -o yaml > install.yaml
```

This will generate the Kubernetes objects yaml files and write them to `install.yaml` which can then be installed to your cluster using:

```shell
kubectl apply -f install.yaml
```
### Local Development Clusters

For local development, Acorn has been tested with Rancher Desktop, Docker Desktop, and Minikube. If you are using one of these systems, please consider the following:

**Rancher Desktop** comes with a working ingress controller, service loadbalancer solution, and storage class by default. No additional configuration is necessary.

**Docker Desktop** comes with a storage class and service loadbalancer solution, but not an ingress. If you're using Docker Desktop and don't have one installed, Acorn will install the [Traefik v2 Ingress Controller](https://doc.traefik.io/traefik/v2.8/providers/kubernetes-ingress/) for you.

**Minikube** comes with a default storage class, but requires that you [enable ingress explicitly](https://kubernetes.io/docs/tasks/access-application-cluster/ingress-minikube/#enable-the-ingress-controller) with the following command:

```shell
minikube addons enable ingress
```

It's not obvious in the above minikube documentation, but after enabling ingress, if you want to access your applications locally, you must also run:

```shell
minikube tunnel
```

The tunnel command also services as the service loadbalancer solution. If it is running, your TCP services will be published to `localhost`.

**K3d** comes with a working ingress controller, service loadbalancer solution, and storage class by default. However, when creating your K3d cluster, you must configure it to proxy traffic from localhost to the cluster, so that endpoints resolve properly:

```shell
k3d cluster create --api-port 6550 -p "80:80@loadbalancer"
```

If you choose to use a port other than `80` like so:

```shell
k3d cluster create --api-port 6550 -p "8081:80@loadbalancer"
```

then you must reflect that in the `acorn install` command by specifying the port with the `--cluster-domain` flag:

```shell
acorn install --cluster-domain '.local.on-acorn.io:8081'
```

**Kind** comes with a working storage class by default, but you need to take some extra steps to get ingress and service loadbalancer capabilities:

- For ingress, you need to configure the `kind` cluster with a host port mapping and then deploy an ingress controller. You can find more details in the [official documentation](https://kind.sigs.k8s.io/docs/user/ingress/).
- For service loadbalancer capabilities, the [`kind` docs](https://kind.sigs.k8s.io/docs/user/loadbalancer/) recommend to deploy [MetalLB](https://metallb.universe.tf/).
