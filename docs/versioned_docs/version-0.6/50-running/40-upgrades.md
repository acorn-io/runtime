---
title: Manual Upgrades
---

When upgrading an Acorn app, you do not need to pass in all arguments on every update. The args are persisted between runs and only the passed in updates are changed.

## What happens during an upgrade

During an upgrade:

1. Container images are updated.
1. Secrets are deployed.
1. Jobs are run.

## Updating a running image

To upgrade the image in production

```shell
acorn update --image [NEW-IMAGE] [APP-NAME]
```

This will replace the Acorn, and if new container images or configurations are provided, the application containers will be restarted.

## Updating parameters

Deployed Acorns can have their parameters changed through the update command. Depending on the parameters being updated it is possible that network connectivity may be lost or containers restarted.

When updating args for the Acorn app, the behavior will be dependent on how the Acorn app was designed/written. Look for documentation from the Acorn app author to understand what is possible and how to operate the Acorn app.

## Updating published DNS names

If an Acorn was deployed like:

```shell
$ acorn run -p my-app.test.example.com:web [IMAGE] --replicas 3 --cluster-mode active-active
purple-field
```

The app DNS can be updated by running:

```shell
acorn update -p my-app.example.com:web purple-field
```

Only the argument being changed needs to be passed in.
