---
title: Structure
---

## The top level keys of an Acorn file

An Acorn file has predefined top level structs:

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

At a minimum, the file needs to specify at least one container to run.

```cue
containers: {
    nginx: {
        image: "nginx"
    }
}
```

### User defined key requirements

Second level keys defined by the user in the `containers`, `volumes`, `secrets`, and `jobs` blocks need to follow these rules:

* contain only lowercase alphanumeric characters, '-' or '.'
* start with an alphanumeric character
* end with an alphanumeric character

Keys defined in `args`, `profiles`, and `localData` should use camel case.
