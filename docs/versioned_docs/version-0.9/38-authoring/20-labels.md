---
title: Labels and Annotations
---

Labels and annotations are Kubernetes constructs for attaching arbitrary metadata as key-value pairs to resources. Often, they are used by third-party integrations to enhance the functionality of Kubernetes. For example, if you were using [cert-manager](https://cert-manager.io/docs/) to provision SSL certificates, you could add the `cert-manager.io/cluster-issuer` annotation to your ingress resources.

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
In the above examples, the core Kubernetes resources created for the acorn container called "frontend" will get the labels and annotations. This includes the deployment, pods, ingress, and services.

You can also specify labels and annotations from the CLI when launching an acorn via the `run` command. See [here](50-running/20-labels.md) for more details.

:::note

If the Acorn installation has [disabled user label and annotation propagation](30-installation/02-options.md#ignoring-user-defined-labels-and-annotations), then, except for the metadata scope, labels and annotations will be silently ignored.

:::

## Metrics

To automatically create Prometheus scrape annotations on your Acorn apps, define the metrics port and HTTP path in the Acornfile:
```acorn
containers: "mycontainer": {
    ports: ["8080/http", "8081/http"]
    metrics: {
        port: 8081
        path: "/metrics"
    }
    // ...
}
```

This would create the following annotations on the Kubernetes Pods for the container:
```yaml
prometheus.io/scrape: "true"
prometheus.io/port: "8081"
prometheus.io/path: "/metrics"
```

The `path` parameter must begin with `/`, and the `port` parameter must be an integer in between 1 and 65535.
