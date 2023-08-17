---
title: "acorn project rm"
---
## acorn project rm

Deletes projects

```
acorn project rm [flags] PROJECT_NAME [PROJECT_NAME...]
```

### Examples

```

acorn project rm my-project

```

### Options

```
  -h, --help   help for rm
```

### Options inherited from parent commands

```
      --config-file string   Path of the acorn config file to use
      --debug                Enable debug logging
      --debug-level int      Debug log level (valid 0-9) (default 7)
      --kubeconfig string    Explicitly use kubeconfig file, overriding the default context
  -o, --output string        Output format (json, yaml, {{gotemplate}})
  -j, --project string       Project to work in
  -q, --quiet                Output only names
```

### SEE ALSO

* [acorn project](acorn_project.md)	 - Manage projects

