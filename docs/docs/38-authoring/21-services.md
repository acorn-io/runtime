---
title: Services
---

## Consuming services

When you are authoring the Acornfile for your application you can define cloud services that will be provisioned for your application. The services will be deployed alongside your application at deploy/run time. To learn how to create your own service Acorns see [services](/100-reference/10-services.md) in the reference section.

### Wiring services into your Acorn app

Service attritibutes are accessed through the `@{}` syntax in the Acornfile. Here is a simple example of accessing the `address` attribute of a service named `db`. For complete service syntax see the [services](/100-reference/03-acornfile.md#services-consuming) section in the Acornfile reference.

```acorn

```acorn
// This service exposes an address and a secret
services: db: {
    image: "ghcr.io/acorn-io/aws/rds-aurora-cluster:latest"
}

containers: app: {
    image: "my-app:latest"
    env: {
        MY_SERVICE_ADDRESS: "@{service.db.address}"
        MY_SERVICE_PORT: "@{service.db.ports.3306}"
        DB_USER: "@{service.db.secrets.admin.username}"
        DB_PASS: "@{service.db.secrets.admin.password}"
        DB_NAME: "@{service.db.data.dbName}"
    }
}
```

In the above example the service db parameters are accessed using the `@{service.db}` syntax. Ports are referenced by the expected value, in this case `3306` for MySQL. However, the actual port number may not be `3306` as it can be dynamically assigned during the service creation.
