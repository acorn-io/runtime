---
title: "acorn uninstall"
---
## acorn uninstall

Uninstall acorn and associated resources

```
acorn uninstall [flags]
```

### Examples

```

# Uninstall with confirmation
acorn uninstall

# Force uninstall without confirmation
acorn uninstall -f
```

### Options

```
  -a, --all     Delete all volumes and secrets
  -f, --force   Do not prompt for confirmation
  -h, --help    help for uninstall
```

### Options inherited from parent commands

```
  -A, --all-projects        Use all known projects
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding current project
  -j, --project string      Project to work in
```

### SEE ALSO

* [acorn](acorn.md)	 - 

