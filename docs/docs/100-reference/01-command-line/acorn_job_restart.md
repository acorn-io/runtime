---
title: "acorn job restart"
---
## acorn job restart

Restart a job

```
acorn job restart [JOB_NAME...] [flags]
```

### Examples

```

acorn job restart app-name.job-name
```

### Options

```
  -h, --help   help for restart
```

### Options inherited from parent commands

```
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding the default context
  -o, --output string       Output format (json, yaml, {{gotemplate}})
  -j, --project string      Project to work in
  -q, --quiet               Output only names
```

### SEE ALSO

* [acorn job](acorn_job.md)	 - Manage jobs

