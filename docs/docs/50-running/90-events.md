---
title: Events
---

Acorn installations maintain a curated event log of the operational details associated with managing applications.

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
```

The command prints events in chronological order, printing the oldest events first.


:::note

`acorn events` does not yet support filtering events, but its output can be piped to secondary tools like `grep` and `jq` to similar effect.

:::

