---
title: "acorn all"
---
## acorn all

List (almost) all objects

```
acorn all [flags]
```

### Examples

```

acorn all
```

### Options

```
  -a, --all             Include stopped apps/containers
  -h, --help            help for all
  -i, --images          Include images in output
  -o, --output string   Output format (json, yaml, {{gotemplate}})
  -q, --quiet           Output only names
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

