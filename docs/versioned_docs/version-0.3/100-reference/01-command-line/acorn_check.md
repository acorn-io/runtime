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
  -A, --all-namespaces      Namespace to work in
      --context string      Context to use in the kubeconfig file
      --kubeconfig string   Location of a kubeconfig file
      --namespace string    Namespace to work in (default "acorn")
```

### SEE ALSO

* [acorn](acorn.md)	 - 

