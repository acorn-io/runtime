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
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding current project
  -j, --project string      Project to work in
```

### SEE ALSO

* [acorn](acorn.md)	 - 

