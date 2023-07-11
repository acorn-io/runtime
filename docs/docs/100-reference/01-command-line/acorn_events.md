---
title: "acorn events"
---
## acorn events

List events about Acorn resources

```
acorn events [flags] [PREFIX]
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

  # Filter by Event Source 
  # If a PREFIX is given in the form '<kind>/<name>', the results of this command are pruned to include
  # only those events sourced by resources matching the given kind and name.
  # List events sourced by the 'hello' app in the current project
  acorn events app/hello
  
  # If the '/<name>' suffix is omitted, '<kind>' will match events sourced by any resource of the given kind.
  # List events related to any app in the current project
  acorn events app 

  # Filter by Event Name
  # If the PREFIX '/<name>' suffix is omitted, and the '<kind>' doesn't match a known event source, its value
  # is interpreted as an event name prefix.
  # List events with names that begin with '4b2b' 
  acorn events 4b2b

  # Get a single event by name
  acorn events 4b2ba097badf2031c4718609b9179fb5

```

### Options

```
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

