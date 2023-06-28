---
title: Linking Acorn Apps
---

Acorn apps can link to containers from other Acorn apps at runtime to provide supporting services. For instance, if you have an Acorn running PostgreSQL, it can be used to provide the `db` service to another app.

If you have an Acorn that defines a `web` container and a `redis` container, you can consume a separate Acorn to provide the redis service from an already running Acorn.

```shell
acorn run --link my-other-redis-acorn:redis [IMAGE]
```

In the above example, the container service from the running Acorn will be available within the new Acorn as `redis`. Your new instance will be able to resolve the `redis` name and it will connect to the remote service defined by the link.

:::note
The port from the container being linked to must be explicitly exposed in the `Acornfile` i.e. `ports: expose: "5432/tcp"`.
:::

The more general linking syntax is as follows:

```shell
acorn run --link <source>:<alias> [IMAGE]
```

`<alias>` is the name that the new Acorn app can use to resolve the linked service.

`<source>` is one of the following:

- The name of another Acorn in the same project, if that Acorn only has one container
- A reference to a particular container in a different Acorn in the same project, in the format `<acorn>.<container>`

For example, if I have an Acorn called `my-app` with two containers `nginx` and `db`, I can link the `db` container to another Acorn in the same project:

```shell
acorn run --link my-app.db:db [IMAGE]
```

:::note
If you set the `<alias>` to a name identical to the name of one of the containers in the new app, then that container in the new app will not be created, since the linked container takes its place.
:::
