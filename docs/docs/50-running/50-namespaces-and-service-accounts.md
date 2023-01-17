---
title: Namespaces and Service Accounts
---
## Namespaces 

By design Acorn will automatically create a new namespace and run apps under this namespace. If you are planning to deploy an app to a pre-created namespace you will need to label it explicitly.

#### To label :

```shell
kubectl label namespaces <pre-created-namespace> acorn.io/app-name=<test-app>
kubectl label namespaces <pre-created-namespace> acorn.io/app-namespace=acorn
```
#### To verify :
```shell
acorn run --target-namespace <pre-created-namespace> -n <test-app> -P ghcr.io/acorn-io/library/hello-world
acorn % kubectl get pods -n ns-test-app
NAME                      READY   STATUS    RESTARTS   AGE
webapp-556947c87d-gt97r   1/1     Running   0          3m54s

```
You can also proivde a namespace that does not exist to the --target-namespace parameter. In this case, Acorn will create the namespace with all the required labels.

:::caution
When the app is removed the namespace will also be deleted.
:::

## Service Accounts

All Kubernetes deployments or jobs launched by Acorn will have a service account attached based on their definition in the Acornfile. 

For example:

```acorn
containers: {
    "my-app-container": {
        // ...
    }
}
jobs: {
    "my-app-job": {
        // ...
    }
}
routers: {
    "my-app-router": {
        // ...
    }
}
```

Running the above Acornfile will result in three sevice accounts named `my-app-container`, `my-app-job`, and `my-app-router` being created.
