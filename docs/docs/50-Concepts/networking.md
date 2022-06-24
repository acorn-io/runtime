---
title: Networking
---

## Overview

Acorn can be used to package applications that can be used stand alone, with other Acorns, and made available to non Acorn based workloads.
Modern day applications are losely coupled over networking paths.

Terminology:

Acorn App - An application that has been defined and packaged within the scope of a single acorn.cue file.

## Acorn App Network scopes

### Internal Acorn communication

When composing an Acorn app that will only need to communicate with itself, you can define the `ports` section on the containers in the app.

```cue
containers: {
   "my-app": {
      build: {
        context: "."
      }
      ports: [
        "4444:4444", // My internal endpoint
      ]
   }
}
```

In the example above other containers within the Acorn App would be able to communicate with `my-app` over port `4444`.

### External Acorn App communications

For services running outside of the Acorn App to communicate with your services, you need to expose the ports. Building on the above example.

```cue
containers: {
   "my-app": {
      build: {
        context: "."
      }
      ports: [
        "4444:4444", // My internal endpoint
      ]
      expose: [
        "80:80", // My site
      ]
   }
}
```

This will make it so workloads on the same cluster can communicate with your Acorn App on port 80.

#### Publishing

If you need to expose your Acorn App to users and workloads outside of your cluster, you will need to publish your services.

By default, all HTTP services are automatically published via the underlying Ingress controller. To publish no ports you can use `-p none`.

// Note this is going to go through a major refactor and likely to change, but the concept holds.

Publishing services is a runtime level decision for the user to make. If a user wants to publish all exposed ports when launching the Acorn App the `-P` flag is used.

```shell
> acorn run -P [APP-IMAGE]
```

In our example this would expose port 80 through the Ingress controller of the underlying Kubernetes cluster.

If the user wants to expose under an explicit name, the user can do the following:

```shell
> acorn run -d my-app.example.com:my-app [APP-IMAGE]
```

That will expose the application under the hostname my-app.example.com. There is no need to pass a publish flag.

To see which services in your Acorn App can be published run `acorn run [APP-IMAGE] --help`

```shell
> acorn run [APP-IMAGE] --help
Volumes:   mysql-data-0, mysql-backup-vol
Secrets:   backup-user-credentials, create-backup-user, user-provided-data, mariadb-0-client-config, mariadb-0-mysqld-config, mariadb-0-galera-config, root-credentials, db-user-credentials
Container: mariadb-0
Ports:     mariadb-0:3306/tcp

      --backup-schedule string         Backup Schedule
      --boot-strap-index int           Set server to boot strap a new cluster. Default (0)
      --cluster-name string            Galera: cluster name
      --custom-mariadb-config string   User provided MariaDB config
      --db-name string                 Specify the name of the database to create. Default(acorn)
      --db-user-name string            Specify the username of db user
      --force-recover                  When recovering the cluster this will force safe_to_bootstrap in grastate.dat for the bootStrapIndex node.
      --recovery                       Run cluster into recovery mode.
      --replicas int                   Number of nodes to run in the galera cluster. Default (1)
      --restore-from-backup string     Restore from Backup. Takes a backup file name
```

Ports that can be exposed are listed under the `Ports` setting.

If you have an app that exposes a TCP endpoint instead of HTTP like:

```cue
containers: {
    ...
    mysql: {
        image: mysql
        expose: "3306:3306"
        ...
    }
    ...
}
```

You can expose these outside the cluster through a loadbalancer endpoint in the following way.

```shell
> acorn run -p 3306:3306 [MY-APP-IMAGE]
```
