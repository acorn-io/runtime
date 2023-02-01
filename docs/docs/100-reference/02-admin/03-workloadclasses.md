---
title: Workload Classes
---
Workload classes are a way of defining scheduling for the applications running on Acorn. They allow you to define Affinities, Tolerations, and Resource Requirements for the Pods that applications will run on.

## Project Workload Classes
A Project Workload Class is associated to a single project. Any apps in that project will have access to the workload class and its configurations. Project Workload Classes in different projects won't interfere with each other, so you can have a Project Workload Class in different projects with the same name and different parameters.

Here is an example of a Project Workload Class with all its configurable fields.
```yaml
kind: ProjectWorkloadClass
apiVersion: admin.acorn.io/v1
default: true # If no class is given for a workload, the default is chosen. Only one default per project.
description: A short description of the workload class
metadata:
  name: workload-class-name
  namespace: project-namespace
memory:
  min: 1Gi
  max: 2Gi
  default: 1Gi # This default overrides the install-wide memory default
  values: # Specific values that are only allowed to be used. Automatically includes the max, min and default.
  - 1.5Gi
cpuScaler: 1 # This is used as a ratio of how many VCPUs to schedule per Gibibyte of memory. In this case it is 1 to 1.
tolerations: # The toleration fields for Pods
  - key: "foo"
    operator: "Equal"
    value: "bar"
    effect: "NoSchedule"
affinity: # The affinity fields for Pods
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: foo
            operator: In
            values:
            - bar
```

If `memory.min`, `memory.max`, `affinity`, and `tolerations` are not given, then there are no scheduling rules for workloads using the workload class. 

## Cluster Workload Classes
Cluster Workload Classes are exactly the same as Project Workload Classes except that they are not namespaced. This means that Cluster Woerkload Classes are available to every app running in your cluster.

Similar to Project Workload Classes, there can be only one default for the entire cluster. However, there can be a default Cluster Workload Class and a default Project Workload Class for any project; the Project Workload Class default will take precedence in this situation. Similarly, if a Cluster Workload Class and a Project Workload Class exist with the same name, then the Project Workload Class will take precedence. These rules are applied when deploying apps and also when using the [`acorn offerings volumeclasses`](100-reference/01-command-line/acorn_offerings_workloadclasses.md) command.