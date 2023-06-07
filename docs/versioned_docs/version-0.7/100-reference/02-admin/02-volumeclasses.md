---
title: Volume Classes
---
Volume classes allow you to define where and how volumes are created. They are an abstraction on top of [storage classes](https://kubernetes.io/docs/concepts/storage/storage-classes/).

By default, Acorn will create volume classes with no restrictions for each storage class in your cluster. These volume classes will be available to every user of your cluster. You can add restrictions to these volume classes as described here.

If you would like to manually control the volume classes, the installation flag [`--manage-volume-classes`](30-installation/02-options.md#manually-managing-volume-classes). Note that using this flag will remove the Acorn generated volume classes.

## Project Volume Classes
A Project Volume Class is associated to a single project. Any apps in that project will have access to the volume class and its underlying storage class. Project Volume Classes in different projects won't interfere with each other, so you can have a Project Volume Class in different projects with the same name and different parameters.

Here is an example of a Project Volume Class with all its configurable fields.
```yaml
kind: ProjectVolumeClass
apiVersion: admin.acorn.io/v1
default: true # If no class is given for a volume, the default is chosen. Only one default per project.
description: A short description of the volume class
metadata:
  name: volume-class-name
  namespace: project-namespace
size:
  min: 1G
  max: 10G
  default: 2G
storageClassName: local-path
allowedAccessModes:
  # List of access modes allowed by this volume class, like:
  - readWriteOnce
  - readWriteMany
inactive: false # An inactive volume class can continue to be used by existing apps, but not by new apps.
```

If `min`, `max`, or `allowedAccessModes` are not given, then there are no restrictions for volumes using the class. If a Project Volume Class does not have a `default` size and a volume does not specify a size, then `10G` is used.

## Cluster Volume Classes
Cluster Volume Classes are exactly the same as Project Volume Classes except that they are not namespaced. This means that Cluster Volume Classes are available to every app running in your cluster.

Similar to Project Volume Classes, there can be only one default for the entire cluster. However, there can be a default Cluster Volume Class and a default Project Volume Class for any project; the Project Volume Class default will take precedence in this situation. Similarly, if a Cluster Volume Class and a Project Volume Class exist with the same name, then the Project Volume Class will take precedence. These rules are applied when deploying apps and also when using the [`acorn offerings volumeclasses`](100-reference/01-command-line/acorn_offerings_volumeclasses.md) command.