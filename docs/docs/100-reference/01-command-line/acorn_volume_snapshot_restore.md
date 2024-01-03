---
title: "acorn volume snapshot restore"
---
## acorn volume snapshot restore

Restore a snapshot to a new volume

```
acorn volume snapshot restore [flags] SNAPSHOT_NAME VOLUME_NAME
```

### Options

```
  -h, --help   help for restore
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

* [acorn volume snapshot](acorn_volume_snapshot.md)	 - Manage snapshots

