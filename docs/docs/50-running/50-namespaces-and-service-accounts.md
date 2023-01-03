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
:::

### Namespace vs Target-Namespace
There are two fields that can be modified when running an Acorn to determine the Acorn's namespace. The global command flag `--namespace` and the ```acorn run``` command flag `--target-namespace`

#### --namespace : 
* Controls the value of label `acorn.io/app-namespace` i.e. where the app and any of its resources are created

#### --target-namespace :
* Controls the namespace where the underlying kubernetes objects get created such as deployments, jobs, pods, volumes, etc

For example take a simple Acornfile,

```acornfile
containers: {
	app1: {
		image: "nginx"
		ports: publish: "80/http"
	}
}
```

and run using

```shell
 acorn --namespace my-namespace run -n proud-dew .
```

this will result in an app named proud-dew to be within the `my-namespace` scope, and it's kubernetes object within the generated `acorn` namespace.

```shell
$ acorn app -A
NAME                     IMAGE          HEALTHY   UP-TO-DATE   CREATED   ENDPOINTS                                                                                                                                  MESSAGE
my-namespace/proud-dew   c9788dc902b7   1         1            14m ago   http://app1-proud-dew-89e87276d4a9.local.on-acorn.io => app1:80       
$ kubectl get pods -A
NAMESPACE            NAME                                     READY   STATUS    RESTARTS       AGE
...
proud-dew-13ae13d6-946   app1-75cc5cfff8-r77ff                    1/1     Running   0              109s
...
```

using the --target-namespace flag

```shell
 acorn run -n proud-dew --target-namespace my-namespace .
```

would result in an app named `proud-dew` to be within the default `acorn` namespace, and it's kubernetes object within the `my-namespace` scope.

```shell
$ acorn app -A
NAME              IMAGE          HEALTHY   UP-TO-DATE   CREATED   ENDPOINTS                                                         MESSAGE
acorn/proud-dew   c9788dc902b7   1         1            22s ago   http://app1-proud-dew-b3b8682a13d9.local.on-acorn.io => app1:80   OK
$ kubectl get pods -A
NAMESPACE            NAME                                     READY   STATUS    RESTARTS       AGE
...
my-namespace         app1-64fcbb8c7-x795r                     1/1     Running   0              24s
...
```

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
