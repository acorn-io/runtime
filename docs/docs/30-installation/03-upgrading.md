---
title: Upgrades
---

## CLI

In order to upgrade Acorn on a Kubernetes cluster you must first download an updated Acorn CLI version.

### Brew

```shell
brew update
brew upgrade acorn-io/cli/acorn
```

### Binary

Download the latest binary version and install following the binary install method.

## Upgrading Acorn on a Kubernetes cluster

Once a new version of Acorn is being used the Acorn version on a Kubernetes cluster will also need to be updated. You can run the following command to do the upgrade:

```shell
acorn install
```

This will download the newest versions of the Acorn components for the cluster.
