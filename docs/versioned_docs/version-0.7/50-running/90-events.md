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

# Print more information about each event
acorn events --details -o yaml
```

The command prints events in reverse chronological order, printing the most recently observed events first.


:::note

`acorn events` does not yet support filtering events, but its output can be piped to secondary tools like `grep` and `jq` to similar effect.

:::

