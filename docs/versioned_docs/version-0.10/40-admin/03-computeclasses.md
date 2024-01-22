---
title: Compute Classes
---
Compute classes are a way of defining scheduling for the applications running on Acorn. They allow you to define Affinities, Tolerations, and Resource Requirements for the Pods that applications will run on.

## Project Compute Classes

A Project Compute Class is associated to a single project. Any apps in that project will have access to the compute class and its configurations. Project Compute Classes in different projects won't interfere with each other, so you can have a Project Compute Class in different projects with the same name and different parameters.

Here is an example of a Project Compute Class with all its configurable fields.

```yaml
kind: ProjectComputeClass
apiVersion: admin.acorn.io/v1
default: true # If no class is given for a workload, the default is chosen. Only one default per project.
description: A short description of the compute class
metadata:
  name: workload-class-name
  namespace: project-namespace
memory:
  min: 1Gi
  max: 2Gi
  default: 1Gi # This default overrides the install-wide memory default
  requestScaler: .5 # Scales the requested memory by this amount to allow over provisioning of memory. Containers will not be able to use more memory then the user requested amount.
  values: # Specific values that are only allowed to be used. Default must be included in these values and max/min cannot be set.
  - 1.5Gi
cpuScaler: 1 # This is used as a ratio of how many VCPUs to schedule per Gibibyte of memory. In this case it is 1 to 1.
resources: # memory and CPU can not be specified in the resources field. 
  limits:
    gpu-vendor.example/example-limit: 1
  requests:
    gpu-vendor.example/example-request: 1
priorityClassName: foo # The priority class to use for Pods
runtimeClassName: bar # The runtime class name to use for Pods
tolerations: # The same toleration fields for Pods
  - key: "foo"
    operator: "Equal"
    value: "bar"
    effect: "NoSchedule"
affinity: # The same affinity fields for Pods
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: foo
            operator: In
            values:
            - bar
supportedRegions: ["local"] # should always be set to ["local"]
```

If `memory.min`, `memory.max`, `memory.values`, `resources`, `affinity`, and `tolerations` are not given, then there are no scheduling rules for workloads using the compute class.

## Cluster Compute Classes

Cluster Compute Classes are exactly the same as Project Compute Classes except that they are not namespaced. This means that Cluster Workload Classes are available to every app running in your cluster.

Similar to Project Compute Classes, there can be only one default for the entire cluster. However, there can be a default Cluster Compute Class and a default Project Compute Class for any project; the Project Compute Class default will take precedence in this situation. Similarly, if a Cluster Compute Class and a Project Compute Class exist with the same name, then the Project Compute Class will take precedence. These rules are applied when deploying apps and also when using the [`acorn offerings volumeclasses`](100-reference/01-command-line/acorn_offerings_computeclasses.md) command.

## Resource provisioning

Compute classes are the primary way to carve up resources in the cluster. When configuring the computeClasses, you should look at the ammount of RAM you have on a host and determine the ratio of CPU to RAM you want to use.

### Example no over provisioning

If you have a 4vCPU machine with 16GiB of RAM you have a 4:16 ratio of CPU to RAM.

If you want to use 1GiB of RAM per vCPU you would set the CPU scalar to .25.

4 vCPU / 16GiB RAM = .25 vCPU per GiB of RAM

### Example over provisioning

Say you want to over provision the resources by 50% to allow higher utilization of the cluster. If you have the same 4vCPU machine with 16GiB of RAM you have a 4:16 ratio of CPU to RAM. You would still set the CPU scalar to .25.

In that configuration you'll be able to run ~16 workloads. Some resources will be unavailable for workloads for system level processes needed to orchestrate the system.

If you would like to be able to run ~32 workloads you would set the resourceScalar field to .5. This value will be used to scale the requested resources by the user.  So if a user requests 1GiB of RAM, the scheduler will only request 512MiB for the workload. The user will only be able to use up to the 1GiB they requested.
