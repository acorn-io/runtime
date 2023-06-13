---
title: "acorn events"
---
## acorn events

List events about Acorn resources

```
acorn events [flags]
```

### Examples

```
# List all events in the current project
  acorn events

  # List events across all projects
  acorn -A events

  # List the last 10 events 
  acorn events --tail 10

  # List the last 5 events and follow the event log
  acorn events --tail 5 -f

  # Getting Details 
  # The 'details' field provides additional information about an event.
  # By default, this field is elided from this command's output, but can be enabled via the '--details' flag.
  # This flag must be used in conjunction with a non-table output format, like '-o=yaml'.
  acorn events --details -o yaml

```

### Options

```
  -d, --details         Don't strip event details from response
  -f, --follow          Follow the event log
  -h, --help            help for events
  -o, --output string   Output format (json, yaml, {{gotemplate}})
  -t, --tail int        Return this number of latest events
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

