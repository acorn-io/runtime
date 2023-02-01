---
title: Scheduling
---

## Memory
Setting `memory` via `acorn run` has the highest order of precedence. When setting this, you operate under the `--memory` flag (`-m` for short).

:::note
Check out the [memory reference documentation](../reference/scheduling#memory) for more information.
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

## Workload Classes
To set a workload class at run time, you can utilize the `--workload-class` flag.

:::note
Check out the [workload class reference documentation](100-reference/06-scheduling.md#scheduling) for more information about how workload classes work.
:::

When setting this value, you have two options - globally or per workload and you can do a combination of both. When setting the memory globally for that Acorn, you just define the workload class you would like to set.

```console
acorn run --workload-class sample foo
```

:::tip
This flag comes with auto completions! Hit tab to see workload classes that can be used.
:::

This will set all workloads in the `foo` acorn to use the `sample` workload class. Adjacently, you can set the memory of each individual workload in the Acorn using a `workload=class` pattern. 

```console
acorn run -m nginx=sample foo
```

This will only update Acorn's `nginx` workload to use the `sample` workload class.

Finally, you can do a combination of both.

```console
acorn run -m sample,nginx=different foo
```

This sets all workloads in the `foo` acorn to use the `sample` workload class except for the `nginx` workload which will have the `different` workload class.
