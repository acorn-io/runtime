---
title: "acorn job"
---
## acorn job

Manage jobs

```
acorn job [flags] [ACORN_NAME|JOB_NAME...]
```

### Examples

```

acorn jobs
```

### Options

```
  -h, --help            help for job
  -o, --output string   Output format (json, yaml, {{gotemplate}})
  -q, --quiet           Output only names
```

### Options inherited from parent commands

```
      --config-file string   Path of the acorn config file to use
      --debug                Enable debug logging
      --debug-level int      Debug log level (valid 0-9) (default 7)
      --kubeconfig string    Explicitly use kubeconfig file, overriding the default context
  -j, --project string       Project to work in
```

### SEE ALSO

* [acorn](acorn.md)	 - 
* [acorn job restart](acorn_job_restart.md)	 - Restart a job

