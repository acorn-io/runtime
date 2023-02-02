---
title: Options
---

## Acorn image
When you install acorn, it will launch several workloads in your cluster, including an api-server and controller. By default, these workloads will use the `ghcr.io/acorn-io/acorn` image. You can customize this image by setting the `--image` option. This is useful if you are installing acorn in an environment where you are required to pull images from a private registry.

## TLS via Let's Encrypt

When you launch an acorn and it has published ports, acorn will generate a unique URL for accessing it, like so:
```bash
$ acorn run -P ghcr.io/acorn-io/library/hello-world

$ acorn ps
NAME       IMAGE          HEALTHY   UP-TO-DATE   CREATED   ENDPOINTS                                                                     MESSAGE
black-sea   ghcr.io/acorn-io/library/hello-world   1         1            6s ago    http://webapp-black-sea-4232beae.qnrzq5.alpha.on-acorn.io => webapp:80      OK
```
By default, endpoints are `http`. To have acorn automatically generate a [Let's Encrypt](https://letsencrypt.org/) certificate and secure your endpoints, you can enable acorn's Let's Encrypt integration like this:
```bash
acorn install --lets-encrypt enabled
```
If you add this flag, you'll be prompted during install to agree to Let's Encrypt's [Terms of Service](https://letsencrypt.org/documents/LE-SA-v1.3-September-21-2022.pdf) and supply an email. You can supply these as flags too:
```bash
acorn install --lets-encrypt enabled --lets-encrypt-tos-agree=true --lets-encrypt-email <your email>
```

:::info
Let's Encrypt integration is only useful if you are running a non-local Kubernetes cluster. If you are running acorn on a local cluster such as Docker Desktop, Rancher Desktop, or minikube, enabling Let's Encrypt will have no effect. We don't issue certificates for the `.local.on-acorn.io` domains that are used in this scenario.
:::

## Endpoint domain names
Acorn provides several installation options for controlling the domain name used to generate endpoints. These are outlined in detail on our [networking page](50-running/02-networking.md#dns).


## Ingress class name
Acorn [requires an ingress controller](01-installing.md#ingress-and-service-loadbalancers) to function properly. If your cluster has more than one ingress controller or if it has one but it isn't set as the [default](https://kubernetes.io/docs/concepts/services-networking/ingress/#default-ingress-class), you can explicitly set the ingress class using `--ingress-class-name`.

## Changing install options
If you want to change your install options after the initial installation, just rerun `acorn install` with the new options. This will update the existing install dynamically.