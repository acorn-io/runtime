---
title: Memory
---
You can configure Acorn apps to have a set memory upon startup.

This allows you to specify memory that an Acorn will abide by when being created. There are a number of ways to set this so that you have full control over defaults, workloads, and maximums. In order of precedence, the ways to set memory are when you:

1. [Run an Acorn](../50-running/55-memory.md)
2. [Author an Acornfile](../38-authoring/03-containers.md#memory)
3. [Install Acorn](../30-installation/02-options.md#memory)

:::note
When installing Acorn, you can also specify `--workload-memory-maximum`. This flag sets a maximum that when exceeded prevents the offending Acorn from being installed.
:::

## Valid memory values
Supported value formats for memory flags include
- 1_234 ->`1234`
- 5M -> `5_000_000`
- 1.5Gi ->`1_610_612_736`
- 0x1000_0000 -> `268_435_456`

These all translate into an exact amount of bytes. We encourage you use the binary representation of large quantities of bytes when interacting with memory such as `Ki`, `Mi`, `Gi`, and `Pi`.

### No limit
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