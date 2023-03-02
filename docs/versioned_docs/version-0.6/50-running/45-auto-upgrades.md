---
title: Automatic Upgrades
---
You can configure Acorn apps to automatically upgrade when a new version of the Acorn image they are using is available.

Automatic upgrade for an app will be enabled if `#`, `*`, or `**` appears in the image's tag as part of the run command. Tags will sorted according to the rules for these special characters described below. The newest tag will be selected for upgrade.

`#` denotes a segment of the image tag that should be sorted numerically when finding the newest tag.

This example deploys the hello-world app with auto-upgrade enabled and matching all major, minor, and patch versions:
```shell
acorn run myorg/hello-world:v#.#.#
```

`*` denotes a segment of the image tag that should be sorted alphabetically when finding the latest tag.

In this example, if you had a tag named alpha and a tag named zeta, zeta would be recognized as the newest:
```shell
acorn run myorg/hello-world:*
```


`**` denotes a wildcard. This segment of the image tag won't be considered when sorting. This is useful if your tags have a segment that is unpredictable.

This example would sort numerically according to major and minor version (ie v1.2) and ignore anything following the "-":

```shell
acorn run myorg/hello-world:v#.#-**
```

NOTE: Depending on your shell, you may see errors when using `*` and `**`. Using quotes will tell the shell to ignore them so Acorn can parse them:
```shell
acorn run "myorg/hello-world:v#.#-**"
```

Automatic upgrades can be configured explicitly via a flag.

In this example, the tag will always be "latest", but acorn will periodically check to see if new content has been pushed to that tag:
```shell
acorn run --auto-upgrade myorg/hello-world:latest
```

To have acorn notify you that an app has an upgrade available and require confirmation before proceeding, set the notify-upgrade flag:
```shell
acorn run --notify-upgrade myorg/hello-world:v#.#.# myapp

```
To proceed with an upgrade you've been notified of:
```shell
acorn update --confirm-upgrade myapp
```

New image versions are checked for on an interval. You can control the default interval via the install command and the the `--auto-upgrade-interval` flag. You can control the interal on a per app basis as part of the run command by specifying the `--interval` flag.
