---
title: "acorn project use"
---
## acorn project use

Set current project

```
acorn project use [flags] PROJECT_NAME
```

### Examples

```

acorn project use acorn.io/my-user/acorn
```

### Options

```
  -h, --help   help for use
```

### Options inherited from parent commands

```
  -A, --all-projects        Use all known projects
      --context string      Context to use in the resolved kubeconfig file
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding current project
      --namespace string    Namespace to work in resolved connection (default "acorn")
  -o, --output string       Output format (json, yaml, {{gotemplate}})
  -j, --project string      Project to work in
  -q, --quiet               Output only names
```

### SEE ALSO

* [acorn project](acorn_project.md)	 - Manage projects

