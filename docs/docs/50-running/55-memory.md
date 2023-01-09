---
title: Memory
---
Setting `memory` via `acorn run` has the highest order of precedence. When setting this, you operate under the `--memory` flag (`-m` for short).

:::note
Check out the [memory reference documentation](../100-reference/06-memory.md) for more information.
:::

### --memory
When setting this value, you have two options - globally or per workload and you can do a combination of both. When setting the memory globally for that Acorn, you just define the memory you would like to set.

```console
acorn run -m 512Mi foo
```

This will set all workloads in the `foo` acorn to have `512Mi` of memory. Adjacently, you can set the memory of each individual workload in the Acorn using a `workload=memory` pattern. 

```console
acorn run -m nginx=512Mi foo
```

This will only update Acorn's `nginx` workload to have `512Mi` of memory.

Finally, you can do a combination of both.

```console
acorn run -m 256Mi,nginx=512Mi foo
```

This sets all workloads in the `foo` acorn to have `256Mi` of memory except for the `nginx` workload which will have `512Mi` of memory.

## No limit
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