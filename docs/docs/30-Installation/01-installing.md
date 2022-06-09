---
title: Installing
---

### Prerequisites
To try out acorn you will need admin access to a Kubernetes cluster. Docker Desktop, Rancher Desktop, and K3s are all great options for trying out acorn for testing/development.

### Install

Download the latest acorn [release](https://github.com/acorn-io/acorn/releases/latest) from the github.

Untar or unzip the downloaded artifact

```shell
#Linux or macOS
tar -xzvf <release>.tar.gz

#test binary (assume local directory)
./acorn 
```

> **Note**: On macOS systems, after extracting the binary from the tar file, you'll be prevented from running the binary from the command line because macOS cannot verify it. You can get it to run by finding the binary in the Finder app, right-clicking it, opening it with Terminal, and then following the prompts to allow a security exception for it to run.
>
> TODO: Fix this once we are able to [do code-signing](https://github.com/acorn-io/acorn/issues/46)


TODO: Document special considerations for installing into the major cloud providers' K8s offerings.