---
title: Removing Acorns
---

When you want to remove an Acorn and its components, you can use the `acorn rm` command.

## Default behavior

To remove an Acorn app run `acorn rm [APP]`. This will remove the app and all of the containers associated with it. If the app has any services, secrets, or volumes they will remain until they are removed manually.  This behavior is to protect against accidental deletion of data.

## Removing services and nested Acorns

To remove services and nested Acorns when removing the Acorn app use the `--all` and optionally `--force` flags. The `--all` flag will remove all services and nested Acorns. The `--force` flag will remove the services and nested Acorns without prompting for confirmation. This will also remove all of the secrets and volumes for the app and the child services and nested Acorns.

Otherwise, you can remove the services and nested Acorns manually with the `acorn rm` command after the app has been removed.

```bash
acorn rm [APP].[SERVICE]
```
