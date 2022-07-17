---
title: Quick Start
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

### Build/Run First Acorn

First you need to initialize your Acorn environment by running:

```shell
> acorn init
```

You will only need to do this once.

Create a new `Acornfile` in your working directory and add the following contents.

```cue
containers: {
 web: {
  image: "nginx"
  expose: "80/http"
  files: {
   // Simple index.html file
   "/usr/share/nginx/html/index.html": "<h1>My First Acorn!</h1>"
  }
 }
}
```

Save the file. What this file defines is a container called *web* based on the nginx container image on Dockerhub. It also declares that port 80 should be exposed and that it will expose an http protocol service. We are also customizing the `index.html` file as part of our packaging process. The contents of the file will be added during the build process.

Now you will need to build your acorn from this file by typing `acorn build .`. This will launch an acorn builder and development registry into your Kubernetes cluster and build the acorn image.

```shell
> acorn build .
[+] Building 2.8s (5/5) FINISHED
 => [internal] load .dockerignore                                                                                       0.0s
 => => transferring context: 2B                                                                                         0.0s
 ...
 => => pushing layers                                                                                                   0.0s
 => => pushing manifest for 127.0.0.1:5000/acorn/acorn:latest@sha256:ec773716b1d180ce4e343cdb4d84736107655401a3d411728  0.0s
346 / 55365718
60d803258f7aa2680e4910c526485488949835728a2bc3519c09f1b6b3be1bb3
```

Now we have a built acorn image identified by the sha (60d803258f7a...) at the end of our build command. To run our acorn app we need to run it.

```shell
> acorn run 60d803258f7a
little-dew
```

Our acorn has started and is named `little-dew`.

To check the status of our app we can run the following.

```shell
> acorn apps little-dew
NAME         IMAGE                                                              HEALTHY   UPTODATE   CREATED              ENDPOINTS                                           MESSAGE
little-dew   60d803258f7aa2680e4910c526485488949835728a2bc3519c09f1b6b3be1bb3   1         1          About a minute ago   http://web.little-dew.local.on-acorn.io => web:80   OK
```

In Chrome or Firefox browsers you can now open the URL listed under the endpoints column.

There is a lot more you can do with an Acorn package. // TODO: see docs for more info.
