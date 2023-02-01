---
title: Labels and Annotations
---

Labels and annotations are Kubernetes constructs for attaching arbitray metadata as key-value pairs to resources. Often, they are used by third-party integrations to enhance the functionality of Kubernetes. For example, if you were using [cert-manager](https://cert-manager.io/docs/) to provision SSL certificates, you could add the `cert-manager.io/cluster-issuer` annotation to your ingress resources.

To allow you to take advantage of such integrations, Acorn supports specifying labels and annotations in your Acornfile. These will be applied to the core Kubernetes resources created by Acorn.

Labels and annotations can be defined as top-level elements in an Acornfile or on individual members of the following resources:
- containers
- jobs
- volumes
- secrets

To define labels or annotations that apply to all resources created for your app, add them as top-level elements:
```acorn
labels: {
    key: "value"
}
annotations: {
    key: "value"
}

containers: {
    // ...
}

volumes: {
    // ..
}
```

To define labels or annotations that apply to a specific resource, add them to the desired resource:
```acorn
containers:{
    frontend: {
        labels: {
            key: "value"
        }
        annotations: {
            key: "value"
        }
    }
    // ...
}
```
In the above examples, the core Kubernetes resources created for the acorn container called "fronted" will get the labels and annotations. This includes the deployment, pods, ingress, and services.

You can also specify labels and annotations from the CLI when launching an acorn via the `run` command. See [here](50-running/20-labels.md) for more details.