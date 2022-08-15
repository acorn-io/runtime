---
title: Linking Acorn Apps
---

Acorns can be linked with other running Acorns at runtime to provide supporting services. For instance if you have an Acorn running Postgresql, it can be used to provide the `db` service to another app.

If you have an Acorn that defines a `web` container and a `redis` container, you can consume a separate Acorn to provide the redis service from an already running Acorn.

```shell
acorn run --link my-other-redis-acorn:redis [IMAGE]
```

In the above example the container service from the running Acorn will be available within the new Acorn as `redis`. Your new instance will be able to resolve the `redis` name and it will connect to the remote service defined by the link.
