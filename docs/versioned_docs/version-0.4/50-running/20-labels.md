---
title: Labels and Annotations
---

As mentioned in the [authoring section](../authoring/labels), you can add labels and annotations to your Acorns that will then be propogated down to the core Kubernetes resources created by Acorn. Authors of Acorn images can add these directly to the Acornfile, but users can also add them at runtime.

The flags for adding labels and annotations allow you to specify the resource type and name you are targeting. This is best explained through examples:

```shell

# Add a label to all resources created by the app
acorn run --label key=value

# Add an annotation to just the top-level acorn app's metadata. No child resources will inherit
acorn run --annotation metadata:key=value

# Add a label to all resources created as part of any containers in the acorn
acorn run --label containers:key=value

# Add an annotation to the resources created as part of a specific container named 'mycontainer' in the acorn
acorn run --annotation containers:mycontainer:key=value
```

Valid resource types are:
- global _(achieved by omitting resource type completely)_
- metadata
- containers
- jobs
- volumes
- secrets

For all resource types except metadata, you can add a name to only apply the label/annotation to the resource matching that name and scope.
