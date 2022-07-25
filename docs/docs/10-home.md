---
title: Home
slug: /
---

> <div style={{"font-family":"Apple Chancery", "font-size": 32}}>Mighty oaks from little acorns grow.</div>
> <br/>
>
> <div style={{"font-family":"Apple Chancery", "font-size": 20}}>~ Old English Proverb, probably a squirrel</div>

:::caution

This site is very much a work in progress. The current structure will change drastically over time. For now, the most useful sections are the [Quick Start](20-quickstart.md) and the [CLI Reference](100-Reference/01-command-line/acorn.md).
:::

## Overview

### Acorn

Acorn is a technology that brings the simplicity of running containers with Docker to Kubernetes. It does this by providing a familiar build, run, and deploy UX to Kubernetes. It provides a DSL to describe your application without the boilerplate of Kubernetes YAML files. With the application described in the Acorn DSL, it builds a portable artifact that contains the manifests and Docker images to run the application.

### What can I use Acorn for?

Acorn can be used to deploy containerized applications, including multi-container apps, onto any Kubernetes cluster, from developer laptop to production clusters in the cloud.

Packaging applications into a single portable artifact that includes all of the dependent OCI images and manifests. By having a single artifact to describe and run your application it makes it easier to move into air-gapped environments.

Running production workloads on the cluster.

Developing applications locally and moving to production without having to switch technology stacks.

## Next steps

* [Installation](/installation/installing)
* [Get Started](/Get%20Started/Running%20an%20Acorn)
* [Developing](/Develop%20with%20Acorn/Develop%20with%20Acorn)
