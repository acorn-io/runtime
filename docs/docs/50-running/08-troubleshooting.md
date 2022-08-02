---
title: Troubleshooting
---

## Viewing logs

To view the logs of your running application you can run:

```shell
acorn logs [APP-NAME]
```

If you would like the logs to continue streaming, you can add `-f` to follow the logs.

## Executing commands inside a container

To execute commands in a running Acorn container, you can do:

```shell
acorn exec [APP-NAME]
```

You will be prompted for which container if there is more than one running.

If you know the container name you can specify it with the `-c` option.

```shell
acorn exec -c web-01 [APP-NAME]
```
