---
title: Best Practices
---

When authoring an Acorn it is best to offer a set of defaults so the user can simply `acorn run [IMAGE]` and be able to try out the Acorn without diving into its documentation first.

### User arguments

Provide the user with `profiles` for common runtime configurations, e.g. for production and testing. Bake in best practices where possible.

Avoid passing sensitive information through the Acorn arguments. Instead define secrets that are auto-generated or bound at runtime with user-defined values.

### Avoid string interpolation for configuration files

It is best to keep configuration to a minimum within the Acornfile. Users should be able to pass in additional configuration and have that merged with the App's internal values.

### Secrets

Allow secrets to be auto-generated by default and kept within the Acorn where possible.

### Images

When adding Acorn to a project, let Acorn handle the building of the Docker image.

When using existing images, prefer official DockerHub library images if available.

Use specific versions and tags of images, at least to the minor version in SemVer. Acorn will package all images at build time and reference them by their SHA, ensuring that the application will always pull the same image. For maintainability and ease of troubleshooting, using a specific version is preferred.

## Upgrades

When dealing with stateful applications, use a unique container per instance. Do **not** use scale for stateful applications. This ensures each instance has a unique and stable FQDN. Scaling up and down is always deterministic.

Each application container should use `dependsOn` for the instance before it. This will ensure that only one application container is taken down at a time.
