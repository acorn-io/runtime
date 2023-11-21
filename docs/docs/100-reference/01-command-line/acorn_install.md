---
title: "acorn install"
---
## acorn install

Install and configure acorn in the cluster

```
acorn install [flags]
```

### Examples

```

acorn install
```

### Options

```
      --acorn-dns string                                  enabled|disabled|auto. If enabled, containers created by Acorn will get public FQDNs. Auto functions as disabled if a custom clusterDomain has been supplied (default auto)
      --acorn-dns-endpoint string                         The URL to access the Acorn DNS service
      --allow-traffic-from-namespace strings              Namespaces that are allowed to send network traffic to all Acorn apps
      --allow-user-annotation strings                     Allow these annotations to propagate to dependent objects, no effect if --ignore-user-labels-and-annotations not true
      --allow-user-label strings                          Allow these labels to propagate to dependent objects, no effect if --ignore-user-labels-and-annotations not true
      --allow-user-metadata-namespace strings             Allow these namespaces to propagate labels and annotations to dependent objects, no effect if --ignore-user-labels-and-annotations not true
      --api-server-cpu string                             The CPU to allocate to the runtime-api-server in the format of <req>:<limit> (example 200m:1000m)
      --api-server-memory string                          The memory to allocate to the runtime-api-server in the format of <req>:<limit> (example 256Mi:1Gi)
      --api-server-pod-annotations stringArray            annotations to apply to acorn-api pods
      --api-server-replicas int                           acorn-api deployment replica count
      --auto-configure-karpenter-dont-evict-annotations   Automatically configure Karpenter to not evict pods with the given annotations if app is running a single replica. (default false)
      --auto-upgrade-interval string                      For apps configured with automatic upgrades enabled, the interval at which to check for new versions. Upgrade intervals configured at the application level cannot be smaller than this. (default '5m' - 5 minutes)
      --aws-identity-provider-arn string                  ARN of cluster's OpenID Connect provider registered in AWS
      --builder-per-project                               Create a dedicated builder per project
      --buildkitd-cpu string                              The CPU to allocate to buildkitd in the format of <req>:<limit> (example 200m:1000m)
      --buildkitd-memory string                           The memory to allocate to buildkitd in the format of <req>:<limit> (example 256Mi:1Gi)
      --buildkitd-service-cpu string                      The CPU to allocate to the buildkitd service in the format of <req>:<limit> (example 200m:1000m)
      --buildkitd-service-memory string                   The memory to allocate to the buildkitd service in the format of <req>:<limit> (example 256Mi:1Gi)
      --cert-manager-issuer string                        The name of the cert-manager cluster issuer to use for TLS certificates on custom domains
      --cluster-domain strings                            The externally addressable cluster domain (default .oss-acorn.io)
      --controller-cpu string                             The CPU to allocate to the runtime-controller in the format of <req>:<limit> (example 200m:1000m)
      --controller-memory string                          The memory to allocate to the runtime-controller in the format of <req>:<limit> (example 256Mi:1Gi)
      --controller-replicas int                           acorn-controller deployment replica count
      --controller-service-account-annotation strings     annotation to apply to the acorn-system service account
      --event-ttl string                                  Amount of time an Acorn event will be stored before being deleted (default '168h' - 7 days)
      --features strings                                  Enable or disable features. (example foo=true,bar=false)
  -h, --help                                              help for install
      --http-endpoint-pattern string                      Go template for formatting application http endpoints. Valid variables to use are: App, Container, Namespace, Hash and ClusterDomain. (default pattern is {{hashConcat 8 .Container .App .Namespace | truncate}}.{{.ClusterDomain}})
      --ignore-user-labels-and-annotations                Don't propagate user-defined labels and annotations to dependent objects
      --image string                                      Override the default image used for the deployment
      --ingress-class-name string                         The ingress class name to assign to all created ingress resources (default '')
      --ingress-controller-namespace string               The namespace where the ingress controller runs - used to secure published HTTP ports with NetworkPolicies.
      --internal-cluster-domain string                    The Kubernetes internal cluster domain (default svc.cluster.local)
      --internal-registry-prefix string                   The image prefix to use when pushing internal images (example ghcr.io/my-org/)
      --lets-encrypt string                               enabled|disabled|staging. If enabled, acorn generated endpoints will be secured using TLS certificate from Let's Encrypt. Staging uses Let's Encrypt's staging environment. (default disabled)
      --lets-encrypt-email string                         Required if --lets-encrypt=enabled. The email address to use for Let's Encrypt registration(default '')
      --lets-encrypt-tos-agree                            Required if --lets-encrypt=enabled. If true, you agree to the Let's Encrypt terms of service (default false)
      --manage-volume-classes                             Manually manage volume classes rather than sync with storage classes, setting to 'true' will delete Acorn-created volume classes
      --network-policies                                  Create Kubernetes NetworkPolicies which block cross-project network traffic (default false)
  -o, --output string                                     Output manifests instead of applying them (json, yaml)
      --pod-security-enforce-profile string               The name of the PodSecurity profile to set (default baseline)
      --profile string                                    The name of the profile to use for the installation. Profiles options are production (prod) and default. (default profile is default)
      --propagate-project-annotation strings              The list of keys of annotations to propagate from acorn project to app namespaces
      --propagate-project-label strings                   The list of keys of labels to propagate from acorn project to app namespaces
      --publish-builders                                  Publish the builders through ingress to so build traffic does not traverse the api-server
      --quiet                                             Only output errors encountered during installation
      --record-builds                                     Keep a record of each acorn build that happens
      --registry-cpu string                               The CPU to allocate to the registry in the format of <req>:<limit> (example 200m:1000m)
      --registry-memory string                            The memory to allocate to the registry in the format of <req>:<limit> (example 256Mi:1Gi)
      --service-lb-annotation strings                     Annotation to add to the service of type LoadBalancer. Defaults to empty. (example key=value)
      --set-pod-security-enforce-profile                  Set the PodSecurity profile on created namespaces (default true)
      --skip-checks                                       Bypass installation checks
      --use-custom-ca-bundle                              Use CA bundle for admin supplied secret for all acorn control plane components. Defaults to false.
      --volume-size-default string                        Set the default size for acorn volumes. Accepts storage suffixes (K, M, G, Ki, Mi, Gi, etc) and "." and "_" separators (default 0)
  -m, --workload-memory-default string                    Set the default memory for acorn workloads. Accepts binary suffixes (Ki, Mi, Gi, etc) and "." and "_" separators (default 0)
      --workload-memory-maximum string                    Set the maximum memory for acorn workloads. Accepts binary suffixes (Ki, Mi, Gi, etc) and "." and "_" separators (default 0)
```

### Options inherited from parent commands

```
      --config-file string   Path of the acorn config file to use
      --debug                Enable debug logging
      --debug-level int      Debug log level (valid 0-9) (default 7)
      --kubeconfig string    Explicitly use kubeconfig file, overriding the default context
  -j, --project string       Project to work in
```

### SEE ALSO

* [acorn](acorn.md)	 - 

