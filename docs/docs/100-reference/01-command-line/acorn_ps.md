---
title: "acorn ps"
---
## acorn ps

List or get apps

```
acorn ps [flags] [ACORN_NAME...]
```

### Examples

```

acorn ps
```

### Options

```
  -a, --all             Include stopped apps
  -A, --all-projects    Include all projects in same Acorn instance as the current default project
  -h, --help            help for ps
  -o, --output string   Output format (json, yaml, {{gotemplate}})
  -q, --quiet           Output only names
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

