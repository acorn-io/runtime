---
title: "acorn container"
---
## acorn container

Manage containers

```
acorn container [flags] [APP_NAME...]
```

### Examples

```

acorn containers
```

### Options

```
  -a, --all             Include stopped containers
  -h, --help            help for container
  -o, --output string   Output format (json, yaml, {{gotemplate}})
  -q, --quiet           Output only names
```

### Options inherited from parent commands

```
  -A, --all-namespaces      Namespace to work in
      --context string      Context to use in the kubeconfig file
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Location of a kubeconfig file
      --namespace string    Namespace to work in (default "acorn")
```

### SEE ALSO

* [acorn](acorn.md)	 - 
* [acorn container kill](acorn_container_kill.md)	 - Delete a container

