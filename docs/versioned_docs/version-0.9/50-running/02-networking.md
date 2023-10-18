---
title: Networking
---

Applications will need to be reachable by their users. This can be users on the internet, within the organization, and other application teams. When deploying the Acorn app, there is a lot of flexibility.

## Defaults

By default Acorn apps run with the Acornfile author's default configurations.

```shell
acorn run registry.example.com/myorg/image
```

### Publish all ports

To publish all ports you can run with `-P` or the long form `--publish-all`.

### Publish no ports

To launch the Acorn app with all ports internal, you can launch with the `-P=false` or `--publish-all=false`.

## Publish individual ports

Publishing ports makes the services available outside of the cluster. HTTP ports will be exposed via the layer 7 ingress and TCP/UDP ports will be exposed via service load balancers. When specifying a port to publish without its protocol, the protocol defined for it in the Acornfile will be used. If no protocol is defined in the Acornfile, the default will be tcp.

| Flag value                  | Description                                                                              |
| --------------------------- | ---------------------------------------------------------------------------------------- |
| `-p 80`                     | Publish port 80 in the Acorn to port 80 with its protocol.                               |
| `-p 80/http`                | Publish port 80/http in the Acorn to a random hostname.                                  |
| `-p 81:80`                  | Publish port 80 in the Acorn to 81 on cluster with the port's protocol.                  |
| `-p 81:80/tcp`              | Publish 80/tcp in the Acorn to 81 as a cluster service.                                  |
| `-p 81:80/http`             | Publish 80/http in the Acorn to 81 with a random hostname.                               |
| `-p app:80`                 | Publish container `app` port 80 in the Acorn to 80 on cluster with its protocol.         |
| `-p app:80/tcp`             | Publish container `app` port 80/tcp in the Acorn to 80 as a cluster service.             |
| `-p app:80/http`            | Publish container `app` port 80/http in the Acorn to 80 with a random hostname.          |
| `-p app.example.com:app`    | Publish container `app` protocol HTTP from the Acorn to external name `app.example.com`. |
| `-p app.example.com:app:80` | Publish container `app` port 80 from the Acorn to external name `app.example.com`.       |

## Expose individual ports

Exposing ports makes the services available to applications and other Acorns running on the cluster. When specifying a port to expose without its protocol the protocol defined for it in the Acornfile will be used. If no protocol is defined in the Acornfile, the default will be tcp.

| Flag value               | Description                                                                               |
| ------------------------ | ----------------------------------------------------------------------------------------- |
| `--expose 80`            | Expose port 80 in the Acorn to port 80 with its protocol.                                 |
| `--expose 81:80`         | Expose 80 in the Acorn to 81 on cluster with its protocol.                                |
| `--expose 81:80/tcp`     | Expose 80/tcp in the Acorn to 81 as a cluster service.                                    |
| `--expose 81:80/http`    | Expose 80/http in the Acorn to 81 as a cluster service.                                   |
| `--expose app:80`        | Expose container `app` port 80 with its protocol in the Acorn to 80 as a cluster service. |
| `--expose app:80/tcp`    | Expose container `app` port 80/tcp in the Acorn to 80 as a cluster service.               |
| `--expose app:80/http`   | Expose container `app` port 80/http in the Acorn to 80  as a cluster service.             |
| `--expose web:80:app:80` | Expose container `app` port 80/tcp as cluster service called `web` on port 80/tcp.        |

## DNS

When an Acorn app has published a port, it will be accessible on a unique endpoint. This endpoint can be seen in the output of `acorn app`:

```shell
NAME             IMAGE                 HEALTHY   UP-TO-DATE   CREATED     ENDPOINTS                                                                           MESSAGE
purple-water     my-org/my-acorn:v1    1         1            50s ago     http://app-purple-water-caa5ade4.local.oss-acorn.io => default:8080   OK
```

You have significant control over the domain name and format of your endpoints, as described below.

### Cluster Domain

Your cluster domain is the domain used as the suffix for all your Acorn apps' endpoints.

If you're running a local cluster, such as Docker Desktop, Rancher Desktop, or Minikube, the cluster domain will be `local.oss-acorn.io` and will always resolve to `localhost`.

If you are running any other type of cluster, Acorn provides a DNS service that will reserve a unique cluster domain and create publicly accessible DNS entries for you. The domain will look like `<random cluster ID>.oss-acorn.io` and will resolve to the hostnames or IP addresses supplied by your ingress controller.  This domain is unique to your cluster and will be used for all Acorn apps.

:::caution

The oss-acorn.io DNS service is a public service ran on the Internet. Your Acorn installation must have outbound access to <https://oss-dns.acrn.io> to access it.

To create and maintain public DNS entries, the DNS service expects your Acorn installation to make on-demand and periodic renewal requests to it.

If your Acorn installation ceases to make requests to the DNS service, your DNS entries and reserved domain will eventually expire and be deleted.
:::

You can choose to use your own cluster domain instead of the generated domain like so:

```shell
acorn install --cluster-domain my-company.com
```

If you do so, you must manage your own DNS entries.

You can turn off the Acorn DNS feature as part of installation (or after by rerunning the command):

```shell
acorn install --acorn-dns disabled
```

This will prevent Acorn from reserving a domain for your cluster. If you disable the DNS service and haven't defined a custom cluster domain, the `local.oss-acorn.io` domain will be used as a fallback.

### Naming Conventions

The endpoints generated for your Acorn apps follow this convention by default:

```
<container name>-<app name>-<unique hash>.<cluster domain>
```

Here's an example:

```
web-purple-water-7961a9e1.73fh5y.oss-acorn.io
```

Let's break that FQDN down:

- **web** is the name of the container from your Acorn app. If the container name is "*default*", it will be omitted from the FQDN.
- **purple-water** is the generated name of your Acorn app. You can control this by supplying a name through the `--name` flag.
- **7961a9e1** is a hash created from the container name and app name together. This ensures that the url is unique and will be persistent across runs.
- **73fh5y.oss-acorn.io** is the cluster domain generated for your cluster. You can control this as described in the previous section.

To highlight the level of control this gives you, consider the following:

In addition to this method of controlling your endpoint, you can publish to an explicit external name using the `--publish` flag. See the [publishing ports](#publish-individual-ports) section for more details.

### Endpoint Patterns

Endpoints generated by Acorn are built using a Go Template. By default, endpoints are built using the `{{.Container}}-{{.App}}-{{.Hash}}.{{.ClusterDomain}}` pattern to allow integration with [Let's Encrypt](30-installation/02-options.md#tls-via-lets-encrypt).

However, should you want your application endpoints to be something completely different, you can pass a Go Template to the `--http-endpoint-pattern` flag. For example, say that we want our `blog` container to be reachable at `blog.local.oss-acorn.io` instead of the the standard Let's Encrypt variant. To accomplish this, we just need to install Acorn with a pattern for it set.

```shell
acorn install --http-endpoint-pattern "{{.Container}}.{{.ClusterDomain}}"
```

This will work for existing installations of Acorn by overwriting the current pattern. To go to the original default, simply reinstall with the flag set to `""` like so.

```shell
acorn install --http-endpoint-pattern ""
```

When building the template, there are a few variables that can be referenced in any order.

| Variable           | Description                                             |
| ------------------ | ------------------------------------------------------- |
| {{.Container}}     | Name of the container being hit with the endpoint.      |
| {{.App}}           | Name of the application.                                |
| {{.Hash}}          | A hash generated from the Container and App together.   |
| {{.Namespace}}     | Namespace that the application is deployed in.          |
| {{.ClusterDomain}} | Domain set during installation with `--cluster-domain`. |

You can arrange these variables in any order as long as they do not break DNS and Kubernetes conventions. Using this you have very close to full control over the endpoint for your http application.

:::caution
There are a few restrictions that will fail validation

1. If any segment (individual string segmented by a `.`) has more than 63 characters, it will not pass validation.
2. You cannot use the root cluster domain as the endpoint pattern alone. (i.e. `{{.ClusterDomain}}`)
:::

## Routers
If you have multiple containers normally they would be exposed as multiple different HTTP services. The router feature allows you to expose those containers as a single HTTP service with separate routes. Let's take a look at how we can achieve this with two containers below.

Starting with a sample Acornfile that exposes two services

```acorn
containers: {
        api: {
                image: "nginx"
                ports: publish: "80/http"
        }
        auth: {
                image: "nginx"
                ports: publish: "80/http"
        }
}
```

If you start this Acornfile with `acorn run` the generated output should look like

```shell
 STATUS: ENDPOINTS[http://api-wild-cloud-a6e8ab1c.local.oss-acorn.io => api:80, http://auth-wild-cloud-aa56b1c9.local.oss-acorn.io => auth:80] HEALTHY[2] UPTODATE[2] OK
```

Adding in the router to the Acornfile

```acorn
routers: myroute: {
    routes: {
        "/auth": "auth:80"
        "/api": {
            pathType: "exact"
            targetServiceName: "api"
            targetPort: 80
        }
    }
}

containers: auth: {
    image: "nginx"
    ports: "80/http"
}

containers: api: {
    image: "nginx"
    ports: "80/http"
}
```

Results in an endpoint that now routes to both services through `/api` and `/auth`

```shell
| STATUS: ENDPOINTS[http://api-delicate-leaf-4ceee54b.local.oss-acorn.io => api:80, http://auth-delicate-leaf-a6e05d96.local.oss-acorn.io => auth:80, http://myroute-delicate-leaf-6633a4ae.local.oss-acorn.io => myroute:8080] HEALTHY[2] UPTODATE[2] OK |
