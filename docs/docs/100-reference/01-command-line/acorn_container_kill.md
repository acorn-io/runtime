---
title: "acorn container kill"
---
## acorn container kill

Delete a container

```
acorn container kill [CONTAINER_NAME...] [flags]
```

### Examples

```

acorn container kill app-name.containername-generated-hash
```

### Options

```
  -h, --help   help for kill
```

### Options inherited from parent commands

```
  -a, --all                 Include stopped containers
  -A, --all-namespaces      Namespace to work in
      --context string      Context to use in the kubeconfig file
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Location of a kubeconfig file
      --namespace string    Namespace to work in (default "acorn")
  -o, --output string       Output format (json, yaml, {{gotemplate}})
  -q, --quiet               Output only names
```

### SEE ALSO

* [acorn container](acorn_container.md)	 - Manage containers

