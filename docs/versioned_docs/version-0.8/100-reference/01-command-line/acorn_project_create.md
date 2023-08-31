---
title: "acorn project create"
---
## acorn project create

Create new project

```
acorn project create [flags] PROJECT_NAME [PROJECT_NAME...]
```

### Examples

```

# Create a project locally
acorn project create my-new-project

# Create a project on remote service acorn.io
acorn project create acorn.io/username/new-project

```

### Options

```
      --default-region string      Default region for project resources
  -h, --help                       help for create
      --supported-region strings   Supported regions for created project
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

