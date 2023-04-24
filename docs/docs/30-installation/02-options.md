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

## Custom CA bundle
Acorn allows a user to provide a custom certificate authority (CA) bundle so that users can add their own private CA that acorn will trust. The most common use case is for acorn to trust an internal image registry that is signed by a private CA.

To do so, you will need to go through the following steps:

1. Provide your CA certificate chain in the following secret.

```bash
kubectl -n acorn-image-system create secret generic cabundle --from-file=ca-certificates.crt=/path/to/your/ca-certificates.crt

kubectl -n acorn-system create secret generic cabundle --from-file=ca-certificates.crt=/path/to/your/ca-certificates.crt
```


:::info
You must provide the **full** CA certificate chain as it will override existing CA certificates in acorn control plane.
:::


2. Install acorn with the following option

```bash
acorn install --use-custom-ca-bundle
```

## Ingress class name
Acorn [requires an ingress controller](30-installation/01-installing.md#ingress-and-service-loadbalancers) to function properly. If your cluster has more than one ingress controller or if it has one but it isn't set as the [default](https://kubernetes.io/docs/concepts/services-networking/ingress/#default-ingress-class), you can explicitly set the ingress class using `--ingress-class-name`.

## Memory
There are two `install` flags for interacting memory: `--workload-memory-default` and `--workload-memory-maximum`. Their values can both be viewed by running `acorn info`.

Check out the [memory reference documentation](100-reference/06-compute-resources.md#memory) for more information.

### --workload-memory-default
This flag is responsible for setting the memory amount that will get defaulted to should no other value be found.

```console 
acorn install --workload-memory-default 512Mi
```

Running the above will set all Acorns on the cluster (current and future) to use `512Mi` as their default memory.

### --workload-memory-maximum
This flag sets a maximum that when exceeded prevents the offending Acorn from being installed.

```console
acorn install --workload-memory-maximum 1Gi
```

This will set it so all Acorns on this cluster will be unable to install should they exceed `1Gi` of memory.

## Ignoring user-defined labels and annotations
There are situations where you may not want a user to be able to label or annotate the objects created by Acorn in the workload cluster. For such circumstances, the installation flag `--ignore-user-labels-and-annotations` exists. If this flag is passed to `acorn install`, then, except for the metadata scope, labels and annotations defined by users in their Acorns will be ignored when creating objects. No error nor warning will be produced.

If this is too restrictive, and you would like to allow certain user-defined labels and annotations to propagate to the Kubernetes objects then you can use the `--allow-user-label` and `allow-user-annotation` installation flags. These flags take a comma-delimited list of label/annotation keys that are allowed to propagate. You can also specify the flags multiple times and the values will be concatenated to create the final list. If the `--ignore-user-labels-and-annotations` is not supplied or is false, then these flags have no effect.

Note that in order to allow propagation of user-defined labels and annotations on an Acorn installation that previous disallowed it, one must pass `--ignore-user-labels-and-annotations=false` to `acorn install`.

## Manually managing volume classes
The default installation of Acorn will automatically create and sync any storage classes in the cluster to volume classes. That means that when a storage class is created or deleted, the corresponding volume class will also be created or deleted. Additionally, the default storage class in the cluster will also become the default volume class. An admin could edit these generated volume classes to set the fields on them (like min/max/default size) and those updates will be maintained. These generated volume classes will be available to every user in the cluster.

If an admin would rather manually manage the volume classes and not have these generated ones, then the `--manage-volume-classes` installation flag is available. The generated volume classes are not generated if this flag is used, and are deleted when the flag is set on an existing Acorn installation. If the flag is again switched off with `--manage-volume-classes=false`, then the volume classes will be generated again.

To recalculate the volume class values for running apps after an admin has modified the values of a volume class, perform the following steps:

1. Run `kubectl edit appinstance <appinstancename> -n <namespace>` to open the configuration file for the app that needs to be updated.
2. Look for the spec section in the configuration file.
3. Add or increment the spec.generation field. Incrementing this field will force the app to recalculate the volume class values using the new values/defaults.
4. Save the changes and exit the editor.

Here's an example of what the relevant part of the AppInstance might look like:

```yaml
spec:
  generation: 3 # increment this value to use new volume class values/defaults
  image: 56b4b08e653a956a4bc36accc74dcc521df17d7b6f23704084a5de562386de6d
```

Note that incrementing the spec.generation field will cause the app to be redeployed with the new volume class values/defaults, which could result in some downtime.

:::note 
The same steps apply for [Compute Classes](50-running/55-compute-resources.md#compute-classes)
:::

## Kubernetes NetworkPolicies
By default, Acorn will automatically create and manage Kubernetes [NetworkPolicies](https://kubernetes.io/docs/concepts/services-networking/network-policies/) to isolate Acorn projects on the network level. This behavior can be disabled by passing `--network-policies=false` to `acorn install`, and can later be re-enabled by passing `--network-policies=true`.

By default, Acorn workloads that publish ports that use HTTP will be allowed to receive traffic from internal (other pods in the cluster) and external (through the cluster's ingress) sources. To secure this further, you can require all traffic to Acorn workloads flow through your ingress by specifying the `--ingress-controller-namespace` parameter during installation.

To allow traffic from a specific namespace to all Acorn apps in the cluster, use `--allow-traffic-from-namespace=<namespace>`. This is useful if there is a monitoring namespace, for example, that needs to be able to connect to all the pods created by Acorn in order to scrape metrics.

## Working with external LoadBalancer controllers
If you are using an external `LoadBalancer` controller that requires annotations on `LoadBalancer` Services to operate, such as the `aws-load-balancer-controller`, you can pass the `--service-lb-annotation` flag to `acorn install`. This will cause Acorn to add the specified annotations to all `LoadBalancer` Services it creates. The value of the flag should be a comma-separated list of key-value pairs, where the key is the annotation name and the value is the annotation value. For example:

```bash
acorn install --service-lb-annotation service.beta.kubernetes.io/aws-load-balancer-type=external,service.beta.kubernetes.io/aws-load-balancer-scheme=internet-facing,service.beta.kubernetes.io/aws-load-balancer-nlb-target-type=instance
```

For readability, you can also pass the flag multiple times, and the values will be concatenated. For example:

```bash
acorn install \
    --service-lb-annotation service.beta.kubernetes.io/aws-load-balancer-type=external \
    --service-lb-annotation service.beta.kubernetes.io/aws-load-balancer-scheme=internet-facing \
    --service-lb-annotation service.beta.kubernetes.io/aws-load-balancer-nlb-target-type=instance \
```

Lastly, you can unset the annotations defined by the `--service-lb-annotation` flag by passing an empty string to the flag. For example:

```bash
acorn install --service-lb-annotation ""
```

:::note
These annotations get added before the the `LoadBalancer` Service is created which is a requisite for some `LoadBalancer` controllers to work properly, like the `aws-load-balancer-controller`.
:::

## Changing install options
If you want to change your installation options after the initial installation, just rerun `acorn install` with the new options. This will update the existing install dynamically.

For strings array flags, you can reset the slice to empty by pass empty string "". For example:

```bash
acorn install --propagate-project-annotation ""
```