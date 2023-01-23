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
      --acorn-dns string                      enabled|disabled|auto. If enabled, containers created by Acorn will get public FQDNs. Auto functions as disabled if a custom clusterDomain has been supplied (default auto)
      --acorn-dns-endpoint string             The URL to access the Acorn DNS service
<<<<<<< HEAD
      --allow-user-annotations strings        Allow these annotations to propagate to dependent objects, no effect if --ignore-user-labels-and-annotations not true
      --allow-user-labels strings             Allow these labels to propagate to dependent objects, no effect if --ignore-user-labels-and-annotations not true
=======
      --annotations-propagation strings       The list of keys of annotations to propagate from acorn project to app namespaces
>>>>>>> 85f920d (Add label and annotation propagation from project)
      --api-server-replicas int               acorn-api deployment replica count
      --auto-upgrade-interval string          For apps configured with automatic upgrades enabled, the interval at which to check for new versions. Upgrade intervals configured at the application level cannot be smaller than this. (default '5m' - 5 minutes)
      --builder-per-project                   Create a dedicated builder per project
      --cluster-domain strings                The externally addressable cluster domain (default .on-acorn.io)
      --controller-replicas int               acorn-controller deployment replica count
      --default-publish-mode string           If no publish mode is set default to this value (default user)
  -h, --help                                  help for install
      --http-endpoint-pattern string          Go template for formatting application http endpoints. Valid variables to use are: App, Container, Namespace, Hash and ClusterDomain. (default pattern is {{hashConcat 8 .Container .App .Namespace | truncate}}.{{.ClusterDomain}})
      --ignore-user-labels-and-annotations    Don't propagate user-defined labels and annotations to dependent objects
      --image string                          Override the default image used for the deployment
      --ingress-class-name string             The ingress class name to assign to all created ingress resources (default '')
      --internal-cluster-domain string        The Kubernetes internal cluster domain (default svc.cluster.local)
      --internal-registry-prefix string       The image prefix to use when pushing internal images (example ghcr.io/my-org/)
      --labels-propagation strings            The list of keys of labels to propagate from acorn project to app namespaces
      --lets-encrypt string                   enabled|disabled|staging. If enabled, acorn generated endpoints will be secured using TLS certificate from Let's Encrypt. Staging uses Let's Encrypt's staging environment. (default disabled)
      --lets-encrypt-email string             Required if --lets-encrypt=enabled. The email address to use for Let's Encrypt registration(default '')
      --lets-encrypt-tos-agree                Required if --lets-encrypt=enabled. If true, you agree to the Let's Encrypt terms of service (default false)
  -o, --output string                         Output manifests instead of applying them (json, yaml)
      --pod-security-enforce-profile string   The name of the PodSecurity profile to set (default baseline)
      --publish-builders                      Publish the builders through ingress to so build traffic does not traverse the api-server
      --record-builds                         Keep a record of each acorn build that happens
      --set-pod-security-enforce-profile      Set the PodSecurity profile on created namespaces (default true)
      --skip-checks                           Bypass installation checks
      --use-custom-ca-bundle                  Use CA bundle for admin supplied secret for all acorn control plane components. Defaults to false.
  -m, --workload-memory-default string        Set the default memory for acorn workloads. Accepts binary suffixes (Ki, Mi, Gi, etc) and "." and "_" seperators (default 0)
      --workload-memory-maximum string        Set the maximum memory for acorn workloads. Accepts binary suffixes (Ki, Mi, Gi, etc) and "." and "_" seperators (default 0)
```

### Options inherited from parent commands

```
  -A, --all-projects        Use all known projects
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding current project
  -j, --project string      Project to work in
```

### SEE ALSO

* [acorn](acorn.md)	 - 

