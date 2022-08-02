---
title: Structure
---

## The top level keys of an Acornfile

An Acornfile has predefined top level structs, and it is recommended to define them in the following order:

```cue
args: { // defines arguments the consumer can provide }
profiles: { // defines a set of default arguments for different deployment types }
containers: { // defines the containers to run the application }
volumes: { // defines persistent storage volumes for the containers to consume }
jobs: { // defines tasks to run on changes or via cron }
acorns: { // other Acorn applications that need to be deployed with your app (databases, etc.) }
secrets: { // defines secret bits of data that are automatically generated or passed by the user }
localData: { // default data and configuration variables }
```

At a minimum, the Acornfile needs to specify at least one container to run.

```cue
containers: {
    nginx: {
        image: "nginx"
    }
}
```

## User defined key requirements

Second-level keys defined by the user underneath the `containers`, `volumes`, `secrets`, and `jobs` blocks must:

* Contain only lowercase alphanumeric characters, `-` or `.`
* Start with an alphanumeric character
* End with an alphanumeric character

Keys defined in `args`, `profiles`, and `localData` should use camelCase.
