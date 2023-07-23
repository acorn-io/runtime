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

  # Filter by Related Resource 
  # If a PREFIX is given in the form '<kind>/<name>', the results of this command are pruned to include
  # only those events related to resources matching the given kind and name.
  # List events related to the 'hello' app in the current project
  acorn events app/hello
  
  # If the '/<name>' suffix is omitted, '<kind>' will match events related to any resource of the given kind.
  # List events related to any app in the current project
  acorn events app 

  # Filter by Event Name
  # If the PREFIX '/<name>' suffix is omitted, and the '<kind>' doesn't match a known event source, its value
  # is interpreted as an event name prefix.
  # List events with names that begin with '4b2b' 
  acorn events 4b2b

  # Get a single event by name
  acorn events 4b2ba097badf2031c4718609b9179fb5

  # Filtering by Time
  # The --since and --until options can be Unix timestamps, date formatted timestamps, or Go duration strings (relative to system time).
  # List events observed within the last 15 minutes 
  acorn events --since 15m

  # List events observed between 2023-05-08T15:04:05 and 2023-05-08T15:05:05 (inclusive)
  acorn events --since '2023-05-08T15:04:05' --until '2023-05-08T15:05:05'

```

### Options

```
  -f, --follow          Follow the event log
  -h, --help            help for events
  -o, --output string   Output format (json, yaml, {{gotemplate}})
  -s, --since string    Show all events created since timestamp
  -t, --tail int        Return this number of latest events
  -u, --until string    Stream events until this timestamp
```

### Options inherited from parent commands

```
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding the default context
  -j, --project string      Project to work in
```

### SEE ALSO

* [acorn](acorn.md)	 - 

