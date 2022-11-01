---
title: Namespaces and Service Accounts
---
## Namespaces 

By design Acorn will run apps under the `acorn` namespace. If you are planning to deploy an app to a pre-created namespace you will need to label it explicitly.

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
:::caution
When the app is removed the namespace will also be deleted.

## Service Accounts

All Kubernetes deployments launched as part of an Acorn will have a service account attached named "acorn"
