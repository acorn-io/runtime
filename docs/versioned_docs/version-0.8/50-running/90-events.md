---
title: Events
---

Acorn installations maintain a curated event log of the operational details associated with managing acorns.

To access this log, use the [`acorn events`](../100-reference/01-command-line/acorn_events.md) command:

```shell
# List all events in the current project
acorn events

# List events across all projects
acorn -A events

# List the last 10 events
acorn events --tail 10

# List the last 5 events and follow the event log
acorn events --tail 5 -f

# List events related to the 'hello' app in the current project
acorn events app/hello
  
# List events related to any app in the current project
acorn events app 

# List events with names that begin with '4b2b' 
acorn events 4b2b

# Get a single event by name
acorn events 4b2ba097badf2031c4718609b9179fb5
```
:::note

Events are printed in chronological order, from oldest to newest, based on the time they were observed.

:::

