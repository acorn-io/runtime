---
title: Structure
---

## Acornfile

### Key ordering

An Acornfile has predefined top level structs, and it is recommended to define them in the following order:

```acorn
name: "// Friendly name of the application"
description: "// A brief description of the application"
readme: "// This is a more descriptive version of the application. Can be pulled in from a file"
info: "// rendered text that describes what the user should do after deployment"
icon: "// a location to an image for the application"
args: { // defines arguments the consumer can provide }
profiles: { // defines a set of default arguments for different deployment types }
services: { // defines services that the application depends on }
containers: { // defines the containers to run the application }
volumes: { // defines persistent storage volumes for the containers to consume }
jobs: { // defines tasks to run on changes or via cron }
acorns: { // other Acorn applications that need to be deployed with your app (databases, etc.) }
secrets: { // defines secret bits of data that are automatically generated or passed by the user }
localData: { // default data and configuration variables }
```

At a minimum, the Acornfile needs to specify at least one container to run.

```acorn
containers: {
    nginx: {
        image: "nginx"
    }
}
```

### Defining conditional blocks

When defining components that are conditionally deployed, use the same ordering as the top level keys within the block.

For example:

```acorn
args: enableRedis: false

containers: {
    nginx: {
        image: "nginx"
        env: {
            DB_PASS: "secrets://db-pass/password"
            DB_USER: "secrets://db-pass/username"
        }
    }
}

if args.enableRedis {
    containers: nginx: env: {
        REDIS_HOST: "redis"
        REDIS_PASS: "secrets://redis-password/token"
    }

    containers: redis: {
        image: "redis"
        env: REDIS_PASSWORD: "secrets://redis-password/token"
    }

    secrets: "redis-password": {
        type: "token"
    }
}

secrets: "db-pass": type: "basic"
```

The above shows how to use an `if` block to add and configure a new container in the Acorn.

### User defined key requirements

Second-level keys defined by the user underneath the `containers`, `volumes`, `secrets`, and `jobs` blocks must:

* Contain only lowercase alphanumeric characters, `-` or `.`
* Start with an alphanumeric character
* End with an alphanumeric character

Keys defined in `args`, `profiles`, and `localData` should use camelCase.
