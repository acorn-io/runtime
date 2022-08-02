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
brew install acorn-io/acorn/acorn
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
