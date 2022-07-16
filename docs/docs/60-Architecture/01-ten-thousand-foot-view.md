---
title: Ten-Thousand Foot View
---

The following is a high-level diagram of Acorn's architecture.

![Architecture](/diagrams/architecture.drawio.svg)

These components are described in more detail below.

### CLI

The acorn cli resides on the end user's machine and executes commands against an instance of acorn running inside a Kubernetes cluster.

### API Server

The acorn API server is a Kubernetes-style API that is made accessible through the Kubernetes [API aggregation layer](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/).

By design, the Acorn CLI only interacts with the `api.acorn.io` API group in Kubernetes. This means that an acorn user only needs RBAC permissions to the API group. The diagram shows the cli talking to the Kubernetes api-server because it is the entrypoint. It handles authentication and routing, but otherwise proxies the request to the acorn API server.

Upon receiving requests from the CLI, the API server will take various actions. Here are a few examples of such actions:

- To launch an app, the API server will create an instance of AppInstance.internal.acorn.io CRD.
- To build an image, where the cli is expecting to have a long-lived interactive connection to the image building service (buildkit), the API server just acts as a proxy.
- To display details about images, it will make requests to internal and external registries.

### Controller

The acorn-controller is responsible for translating acorn "apps" into concrete Kubernetes resources such as deployments, services, and persistentVolumes. It handles the entire lifecycle of such applications and ensures the Kubernetes resources remain in sync with the application.

### Buildkit and Internal Registry

The image build service, buildkit, and an internal image registry are deployed as sibling containers in a single pod. This simplifies the communication between the two components when buildkit is building new acorn images.
