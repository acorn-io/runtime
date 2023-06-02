---
title: Services
---

Service Acorns offer a convenient way for developers to provision databases, caches, storage buckets, and other as-a-service offerings from cloud providers and Kubernetes operators from within the Acorn context. The provisioning process can be implemented using any tooling that can be called from a container.

## Creating a service Acorn

To create a Service Acorn, you will need to create an Acornfile that interacts with the required provider APIs for creating, updating, and deleting the desired service. It will then need to generate a new Acornfile providing the connection details of the created service.

Follow these steps to create a Service Acorn:

1. Declare that a service is going to be created in this Acornfile.

    ```acorn
    services: "my-service": {
        generated: job: "create-service"
    }
    ```

    The snippet above specifies that the complete service definition will be coming from a job called `create-service`. This is similar to how generated secrets are created.

1. Declare any secrets as type generated that will be used by consumers of the service.

    Typically credentials or API keys will be required to connect to the provisioned endpoints. If these are to be exposed to the consuming Acorn app, they need to be declared as generated secrets in the service Acornfile.

    ```acorn
    services: "my-service": {
        generated: job: "create-service"
    }
    // ...
    secrets: "my-service-secret": {
        type: "generated"
        params: job: "create-service"
    }
    ```

    The above snippet adds a secret `my-service-secret` that must be produced by the `create-service` job.

1. Last, create the job(s) that will be used to manage the lifecycle of the service. The job must produce a complete Acornfile that defines the complete service and any expected secrets. The output must be written to `/run/secrets/output` the same as a generated secret. The job can use any language or tooling to create the generated Acornfile. These examples use a shell script for simplicity and readability but could be generated in other languages.

    ```acorn
    services: "my-service": {
      generated: job: "create-service"
    }
    jobs: "create-service": {
        image: "ubuntu:latest"
        entrypoint: "/app/create-service.sh"
        files: "/app/create-service.sh": """
          #!/bin/bash
          cat <<EOF > /run/secrets/output
          services: "my-service": {
            secrets: ["my-service-secret"]
          }
          secrets: "my-service-secret": {
              type: "basic"
              data: {
                username: "foo"
                password: "bar"
              }
          }
          EOF
          """
    }
    secrets: "my-service-secret": {
        type: "generated"
        params: job: "create-service"
    }
    ```

    The example above shows a job that creates a service that exposes a secret containing basic auth credentials. Here it is static, but it could be generated dynamically by the job reaching out to a service like AWS secret manager or Vault.

   The Acornfile here is using an inline script for illustration, the best practice is to use a separate file and copy it into the container like the following example.

    ```acorn
    services: "my-service": {
        generated: job: "create-service"
    }
    jobs: "create-service": {
        image: "ubuntu:latest"
        entrypoint: "/app/create-service.sh"
        dirs: "/app/create-service.sh": "./create-service.sh"
        events: ["create", "update", "delete"]
    }
    secrets: "my-service-secret": {
        type: "generated"
        params: job: "create-service"
    }
    ```

    In this case the `create-service.sh` file will be copied into the container at `/app/create-service.sh`, it is easier to maintain scripts this way. Using this method, you will need to ensure the script is executable.

## Lifecycle

To automate the lifecycle of the service, there are lifecycle events that can be used to trigger the correct behavior. The events are:

- `create` - Runs on the initial deployment of the Acorn.
- `update` - This will be called when the Acorn app is updated or restarted.
- `stop`   - This event will be called when the Acorn app is stopped by the command `acorn stop [APP_NAME]`.
- `delete` - This will be called when the Service Acorn is being deleted.

If no events are specified the default behavior is run on create and update.

If no job is watching for the delete event, the resources created outside of Acorn will be orphaned when the Service Acorn is deleted.

## Troubleshooting

### Generated secret contains the generated Acornfile instead of secret values

In this case, it means that there is an issue with the formatting of the generated output. Common issues are missing double quotes around names with hyphens. A quick way to determine what might be wrong is to copy the contents to a scratch `Acornfile` and run `acorn render .` to see if there are formatting issues.
