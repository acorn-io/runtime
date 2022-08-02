---
title: Ten-Thousand Foot View
---

The following is a high-level diagram of Acorn's architecture.

![Architecture](/diagrams/architecture.drawio.svg)

These components are described in more detail below.

### CLI

The Acorn CLI resides on the end user's machine and executes commands against an instance of Acorn running inside a Kubernetes cluster.

### API Server

The Acorn API server is a Kubernetes-style API that is made accessible through the Kubernetes [API aggregation layer](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/).

By design, the Acorn CLI only interacts with the `api.acorn.io` API group in Kubernetes. This means that an Acorn user only needs RBAC permissions to the API group. The diagram shows the CLI talking to the Kubernetes API server because it is the entrypoint. It handles authentication and routing, but otherwise proxies the request to the Acorn API server.

Upon receiving requests from the CLI, the Acorn API server will take various actions. Here are a few examples of such actions:

- To launch an Acorn app, the Acorn API server will create an instance of the `AppInstance.internal.acorn.io` CRD.
- To build an Acorn image, where the CLI is expecting to have a long-lived interactive connection to the image building service (which is Buildkit), the Acorn API server just acts as a proxy.
- To display details about Acorn images, it will make requests to internal and external registries.

### Controller

The Acorn Controller is responsible for translating Acorn apps into actual Kubernetes resources such as Deployments, Services, and PersistentVolumes. It handles the entire lifecycle of such applications and ensures that the Kubernetes resources remain in sync with the Acorn app definition.

### Buildkit and Internal Registry

The image building service, Buildkit, and an internal image registry are deployed as sibling containers in a single pod. This simplifies the communication between the two components when Buildkit is building new Acorn images.
