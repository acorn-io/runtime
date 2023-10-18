---
title: Compute resources
---

## Memory
Setting `memory` via `acorn run` has the highest order of precedence. When setting this, you operate under the `--memory` flag (`-m` for short).

:::note
Check out the [memory reference documentation](100-reference/06-compute-resources.md#memory) for more information.
:::

### --memory
When setting this value, you have two options - globally or per workload and you can do a combination of both. When setting the memory globally for the Acorn, you provide the memory you would like to set.

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

## Compute Classes
To set a compute class at run time, you can utilize the `--compute-class` flag.

:::note
Check out the [compute class reference documentation](100-reference/06-compute-resources.md#compute-classes) for more information about how compute classes work.
:::

### --compute-class

When setting compute classes, you have two options - globally or per workload, and you can do a combination of both. When setting the compute class globally for the Acorn, you provide the compute class you would like to set. Any classes defined for a specific container or job will overwrite the global value.

```console
acorn run --compute-class sample foo
```

This will set all workloads in the `foo` acorn to use the `sample` compute class. Adjacently, you can set the memory of each individual workload in the Acorn using a `workload=class` pattern. 

```console
acorn run --compute-class nginx=sample foo
```

:::tip
This flag comes with auto completions! Hit tab to see compute classes that can be used.
:::

This will only update Acorn's `nginx` workload to use the `sample` compute class.

Finally, you can do a combination of both.

```console
acorn run --compute-class sample,nginx=different foo
```

This sets all workloads in the `foo` acorn to use the `sample` compute class except for the `nginx` workload which will have the `different` compute class.
