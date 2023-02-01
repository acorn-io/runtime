---
title: Scheduling
---

Acorn comes with a number of features that allow you define how your application workloads will be scheduled at deploy-time.

## Memory
You can configure Acorn apps to have a set memory upon startup.

This allows you to specify memory that an Acorn will abide by when being created. There are a number of ways to set this so that you have full control over defaults, workloads, and maximums. In order of precedence, the ways to set memory are when you:

1. [Run an Acorn](../running/scheduling)
2. [Author an Acornfile](../authoring/containers#memory)
3. [Install Acorn](../installation/options#memory)

:::note
When installing Acorn, you can also specify `--workload-memory-maximum`. This flag sets a maximum that when exceeded prevents the offending Acorn from being installed.
:::

### Valid memory values
Supported value formats for memory flags include
- 1_234 ->`1234`
- 5M -> `5_000_000`
- 1.5Gi ->`1_610_612_736`
- 0x1000_0000 -> `268_435_456`

These all translate into an exact amount of bytes. We encourage you use the binary representation of large quantities of bytes when interacting with memory such as `Ki`, `Mi`, `Gi`, and `Pi`.

#### No limit
It is possible to set any of these flags to have no limit on memory by simply setting their value to `0`. However, running an Acorn with its memory set to `0` while the `--workload-memory-maximum` is not set to `0` will roughly translate to "use as much memory as allowed". For example, say that we install Acorn with a non-zero `--workload-memory-maximum`.

```console
acorn install --workload-memory-maximum 512Mi
```

Then we try to run an Acorn with its memory set to 0.

```console
acorn run --memory 0 foo
```

When the `foo` Acorn gets provisioned, all of its containers will have their memory set to `512Mi` (the `--workload-memory-maximum` we set prior).

:::note
This same interaction will occur if the `--workload-memory-default` is set to 0 (which it is by default)
:::

## Workload Classes
You can configure Acorn apps to have a set workload class upon startup.

Setting a workload class allows you to define what the infrastructure providing your Acorn workloads should look like. Things that workload classes control include:

- What OS/Architecure your workloads will run on
- How much memory is minimal, maximal, default and allowed
- How many vCPUs should be allocated

:::info
You are not able to set vCPUs directly. This is an intentional abstraction and instead vCPUs are calculated off of the amount of memory for a workload.
:::

### Using a Workload Class
You can see the workload classes available in your current project using the CLI. 

```console
$ acorn offerings workloadclasses
NAME          DEFAULT   MEMORY RANGE      MEMORT DEFAULT   DESCRIPTION         
default       *         512Mi-1Gi         1Gi              Default WorkloadClass
non-default             0-1Gi             512Mi            Non-default WorkloadClass
unrestricted            Unrestricted      512Mi            Unrestricted WorkloadClass
specific                128Mi,512Mi,1Gi   128Mi            Specific WorkloadClass
```

Breaking this down, `MEMORY_DEFAULT` tells us what memory we will get if we don't specify any. `MEMORY_RANGE` tells us what memory values are available to use. If it is a range, specified with a `-` then you can use any value in that range. If it has specific values, denoted by commands, then you can only use those values.

Specify workloads classes can be done in the Acornfile (using the `class` property for containers) or at runtime (using the `--workload-class` flag). 

If you do not specify a workload class, the default workload class for the project will be used. If there is no default for the project, the default for the cluster will be used. Finally, if there is no cluster default then no workload class will be used. Depending on the workload class that is used, the memory that you specify may be in contention with its requirements. Should they happen Acorn will provide a descriptive error message to ammend any issues.

:::note
Looking to manage a workload class? This should only be done if you are (or are in communication with) an administrator of Acorn. You can read more information about managing workload classes [here](./02-admin/03-workloadclasses.md)
:::