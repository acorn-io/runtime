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
