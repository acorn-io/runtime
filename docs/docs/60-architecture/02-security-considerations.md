---
title: Security Considerations
---

## Report a Vulnerability

Please email security@acorn.io to report a security vulnerability.

You may optionally use GPG key `ed25519/2B480FBF4589035A` to encrypt the message:
```
-----BEGIN PGP PUBLIC KEY BLOCK-----

mDMEYumxgxYJKwYBBAHaRw8BAQdAOKLnuP8xUhoLv3W9Pt1dm9vHugQp6wfvAOXp
fdsec6y0IkFjb3JuIFNlY3VyaXR5IDxzZWN1cml0eUBhY29ybi5pbz6IkwQTFgoA
OxYhBA6PZhQ3ctoS4HilaitID79FiQNaBQJi6bGDAhsDBQsJCAcCAiICBhUKCQgL
AgQWAgMBAh4HAheAAAoJECtID79FiQNasGcA/0iK+CvUt4WR4EJQCat+59kDga4W
q93NusAccHpVmS2BAP91JcJ8OwZyV8IcW/dwcDFvBvH3A5R0BMejEYLeP+goBbg4
BGLpsYMSCisGAQQBl1UBBQEBB0D+8r9ML4HBgrobV+rysJQ205UKIFmbgVTY5bmD
ywWsRAMBCAeIeAQYFgoAIBYhBA6PZhQ3ctoS4HilaitID79FiQNaBQJi6bGDAhsM
AAoJECtID79FiQNagEEA/2XN7UI70hZNVcm+kGTW04MQ1s57YVzwibGcB79AvIX+
AP9aDhMMlbAOJTVFlBmkzQ6ERpCfakLvw+BM2hEv10uiAg==
=B32Z
-----END PGP PUBLIC KEY BLOCK-----
```

## Acorn System Access

Acorn system components run with cluster admin privileges because it needs the ability to create namespaces and other objects on the user's behalf. End users have little required permissions.

## User Tenancy

Acorn allows multiple teams to deploy and manage Acorn apps on a cluster without interfering with each other.

### Scope

The unit of tenancy is the Acorn namespace, the default one is `acorn`. A user with access to that namespace will be able to see all Acorn apps running in that environment. They will be able to access the logs, containers, and endpoints.

All Acorn CLI commands and the UI are scoped to the user's Acorn namespace.

### RBAC

Users will require CRUD access to all types from the `api.acorn.io` API group within their Acorn namespace.

```shell
"apps"
"apps/log"
"builders"
"builders/port"
"builders/registryport"
"images"
"images/tag"
"images/push"
"images/pull"
"images/details"
"volumes"
"containerreplicas"
"containerreplicas/exec"
"credentials"
"secrets"
"secrets/reveal"
"infos"
```

Optionally, they might need access to create secrets and possibly CertManager objects for TLS certs in the users Acorn namespace. This is if the app team running the Acorn app will be creating secrets to pass in data.

Users can be given access to multiple Acorn namespaces and will then be able to switch between them from the CLI.

For convenience, the following roles are created by the Acorn installation, and can be used to give users access to the resources necessary to perform the corresponding tasks:
```shell
acorn:cluster:edit
acorn:cluster:view
acorn:project:admin
acorn:project:build
acorn:project:edit
acorn:project:view
acorn:project:view-logs
```

### Shared Image Registry

Default installations of Acorn Runtime will deploy an OCI registry into Kubernetes, which will be the default image storage for all projects.
This registry is not secured with credentials.
Acorn's API server performs basic checks to try to prevent users from trivially accessing images that belong to other projects, but this is not guaranteed.

You can configure Acorn to use an external OCI registry, but the credentials are shared between all projects, so the security implications around image access are the same.

## Credentials

Credentials refer to credentials used to pull from and/or push to OCI registries.
In the future credentials in Acorn may be used for different types of services, but at the moment they are only used for OCI registries where Acorn images are stored.

### Storage

Credentials are stored within the cluster in a namespaced secret.
The Acorn API does not give access to the secret values of the credential, namely the password or token.
If a user has access to use the credential that does not mean they can see the credential value.
This makes it safe to share credentials in a team setting.

### Scope/Access

Credentials are valid for all Acorn apps and Acorn images in a namespace.
Any user that has privileges to push or pull and Acorn image will implicitly be using the credentials stored in that namespace.
Similarly any Acorn app that is deployed will use the credentials available in the namespace to pull the Acorn image and referenced Docker images.

### CLI

Credentials are managed with the [`acorn credential`](100-reference/01-command-line/acorn_credential.md) command.

## Networking

### Acorn app Network scopes

Acorn can be used to package applications that can be used standalone, with other Acorns, and made available to non Acorn based workloads.
Modern day applications are loosely coupled over networking paths.

Terminology:

- Acorn image - An application with all its resources, dependencies and configuration, defined by a single Acornfile and packaged as an OCI image
- Acorn app - An instantiation of an Acorn image

### Internal Acorn communication

When composing an Acorn app that will only need to communicate with itself, you can define the `ports` section on the containers in the app.

```acorn
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

In the example above other containers within the Acorn app would be able to communicate with `my-app` over port `4444`.

### External Acorn app communications

For services running outside of the Acorn app to communicate with your services, you need to expose the ports. Building on the above example.

```acorn
containers: {
   "my-app": {
      build: {
        context: "."
      }
      ports: {
        internal: "4444:4444", // My internal endpoint
        expose: "80:80", // My site
      }
   }
}
```

This will make it so workloads on the same cluster can communicate with your Acorn app on port 80.

### Publishing

If you need to expose your Acorn app to users and workloads outside of your cluster, you will need to publish your services.

By default, all HTTP services are automatically published via the underlying Ingress controller. To publish no ports you can use `-p none`.

:::caution

Note this is going to go through a major refactor and likely to change, but the concept holds.
:::

Publishing services is a runtime level decision for the user to make. If a user wants to publish all exposed ports when launching the Acorn app the `-P` flag is used.

```shell
acorn run -P [APP-IMAGE]
```

In our example this would expose port 80 through the Ingress controller of the underlying Kubernetes cluster.

If the user wants to expose under an explicit name, the user can do the following:

```shell
acorn run -p my-app.example.com:my-app [APP-IMAGE]
```

That will expose the application under the hostname `my-app.example.com`. There is no need to pass a publish flag.

To see which services in your Acorn app can be published run `acorn run [APP-IMAGE] --help`

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

```acorn
containers: {
    // ...
    mysql: {
        image: mysql
        ports: expose: "3306:3306"
        // ...
    }
    // ...
}
```

You can expose these outside the cluster through a loadbalancer endpoint in the following way.

```shell
> acorn run -p 3306:3306 [MY-APP-IMAGE]
```
