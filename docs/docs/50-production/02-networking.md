---
title: Networking
---

Applications will need to be reachable by their users. This can be users on the internet, within the organization, and other application teams. When deploying the Acorn app, there is a lot of flexibility.

### Default

By default Acorn apps run with the Acorn author's default configuration.

`acorn run registry.example.com/myorg/image`

### Selectively publish ports

When publishing specific ports, the operator will need to specify all ports to be published. Any unspecified port will default to internal and be unreachable outside of the Acorn.

To learn which ports are available to publish, look at the image help.

`acorn run registry.example.com/myorg/image --help`

There will be a line `Ports: ...` that outlines the ports. To expose the port run

`acorn run -p 3306:3306/tcp registry.example.com/myorg/image`

### Publish HTTP Ports

To publish an HTTP port on a domain name, you use the `-p` option on the run subcommand.

`acorn run -p my-app.example.com:frontend registry.example.com/myorg/image`

<!-- TODO: add --publish-all -->
<!-- TODO: what about --expose? -->

### DNS
When an Acorn app has published a port, it will be accessible on a unique endpoint. This endpoint can be seen in the output of `acorn app`:

```shell
NAME             IMAGE                 HEALTHY   UP-TO-DATE   CREATED     ENDPOINTS                                                                           MESSAGE
purple-water     my-org/my-acorn:v1    2         2            50s ago   http://purple-water.local.on-acorn.io => default:8080   OK
```

You have significant control over the domain name in your endpoints, as described below.

#### Cluster Domain
Your cluster domain is the domain used as the suffix for all your Acorn apps' endpoints.

If you're running a local cluster, such as Docker Desktop, Rancher Desktop, or Minikube, the cluster domain will be `local.on-acorn.io` and will always resolve to `localhost`.

If you are running any other type of cluster, Acorn provides a DNS service that will reserve a unique cluster domain and create publicly accessible DNS entries for you. The domain will look like `<random cluster ID>.on-acorn.io` and will resolve to the hostnames or IP addresses supplied by your ingress controller.  This domain is unique to your cluster and will be used for all Acorn apps.

:::caution

The on-acorn.io DNS service is a public service ran on the Internet. Your Acorn installation must have outbound access to https://dns.acrn.io to access it.

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
This will prevent Acorn from reserving a domain for your cluster. If you disable the DNS service and haven't defined a custom cluster domain, the `local.on-acorn.io` domain will be used as a fallback.

#### Naming Conventions
The endpoints generated for your Acorn apps follow this convention:
```
<container name>.<app name>.<namespace>.<cluster domain>
```
Here's an example:
```
web.purple-water.my-namespace.73fh5y.on-acorn.io
```
Let's break that FQDN down:
- **web** is the name of the container from your Acorn app. If the container name is "*default*", it will be omitted from the FQDN.
- **purple-water** is the generated name of your Acorn app. You can control this by supplying a name through the `--name` flag.
- **my-namespace** is the namespace you deployed your Acorn app into. By default, we use the `acorn` namespace. If you are using this default, it is omitted from the FQDN.
- **73fh5y.on-acorn.io** is the cluster domain generated for your cluster. You can control this as described in the previous section.

To highlight the level of control this gives you, consider the following:

If you set your cluster domain to `my-company.com`, deployed into the default `acorn` namespace, named your app `blog`, and the container name in the Acornfile was `default`, the published endpoint for your app would be: `http://blog.my-company.com`

In addition to this method of controlling your endpoint, you can publish to an explicit external name using the `--publish` flag. See the ports section for more details.