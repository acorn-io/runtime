---
title: Security Considerations
---

## Acorn System Access

Acorn system components run with cluster admin privileges because it needs the ability to create namespaces and other objects on the user's behalf. End users have little required permissions.

## User Tenancy

Acorn allows multiple teams to deploy and manage Acorn apps on a cluster without interfering with each other.

### Scope

The unit of tenancy is the Acorn namespace, the default is `acorn`. A user who access to that namespace will be able to see all Acorn apps running in that environment. They will be able to access the logs, containers, and endpoints.

All Acorn CLI commands and the UI are scoped to the users Acorn namespace.

### RBAC

Uses will require access to CRUD AppInstance types from v1.acorn.io API group.

Optionally, they might need access to create secrets and possibly CertManager objects for TLS certs. This is if the app team running the Acorn app will be creating secrets to pass in data.

Users can be given access to multiple Acorn namespaces, and will be able to switch between them from the CLI.

## Credentials

Credentials refer to credentials used to pull from and/or push to OCI
registries. In the future credentials in Acorn may be used for different
types of credential, but as it stand they are only used for OCI registries.

### Storage

Credentials are store within the cluster in a namespaced secret. Acorn
API does not give access to the secret values of the credential, namely
the password or token. If a user has access to use the credential that
does not mean they can see the credential value. This makes it safe
to share credentials in a team setting.

### Scope/Access

Credentials are valid for all apps and images in a namespace. Any use
that has privileges to push or pull and image will implicitly be using
the credentials stored in that namespace. Similarily any app that is
deploy will use the credentials available in the namespace to pull the
Acorn image and referenced Docker images.

### CLI

Credentials are managed with the [acorn credential](../100-Reference/01-command-line/acorn_credential.md) command.

## Networking

### Acorn App Network scopes

Acorn can be used to package applications that can be used stand alone, with other Acorns, and made available to non Acorn based workloads.
Modern day applications are loosely coupled over networking paths.

Terminology:

Acorn App - An application that has been defined and packaged within the scope of a single Acornfile file.

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

### Publishing

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
