---
title: "acorn check"
---
## acorn check

Check if the cluster is ready for Acorn

```
acorn check [flags]
```

### Examples

```

acorn check
```

### Options

```
  -h, --help                        help for check
  -i, --image string                Override the image used for test deployments.
      --ingress-class-name string   Specify ingress class used for tests
  -o, --output string               Output format (json, yaml, {{gotemplate}})
  -q, --quiet                       No Results. Success or Failure only.
  -n, --test-namespace string       Specify namespace used for tests
```

### Options inherited from parent commands

```
  -A, --all-projects        Use all known projects
      --context string      Context to use in the resolved kubeconfig file
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding current project
      --namespace string    Namespace to work in resolved connection (default "acorn")
  -j, --project string      Project to work in
```

### SEE ALSO

* [acorn](acorn.md)	 - 

