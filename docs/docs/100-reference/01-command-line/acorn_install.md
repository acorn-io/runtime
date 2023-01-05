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
      --api-server-replicas int               acorn-api deployment replica count
      --auto-upgrade-interval string          For apps configured with automatic upgrades enabled, the interval at which to check for new versions. Upgrade intervals configured at the application level cannot be smaller than this. (default '5m' - 5 minutes)
      --cluster-domain strings                The externally addressable cluster domain (default .on-acorn.io)
      --controller-replicas int               acorn-controller deployment replica count
      --default-publish-mode string           If no publish mode is set default to this value (default user)
  -h, --help                                  help for install
      --http-endpoint-pattern string          Go template for formatting application http endpoints. Valid variables to use are: App, Container, Namespace, Hash and ClusterDomain. (default pattern is {{.Container}}-{{.App}}-{{.Hash}}.{{.ClusterDomain}})
      --image string                          Override the default image used for the deployment
      --ingress-class-name string             The ingress class name to assign to all created ingress resources (default '')
      --internal-cluster-domain string        The Kubernetes internal cluster domain (default svc.cluster.local)
      --lets-encrypt string                   enabled|disabled|staging. If enabled, acorn generated endpoints will be secured using TLS certificate from Let's Encrypt. Staging uses Let's Encrypt's staging environment. (default disabled)
      --lets-encrypt-email string             Required if --lets-encrypt=enabled. The email address to use for Let's Encrypt registration(default '')
      --lets-encrypt-tos-agree                Required if --lets-encrypt=enabled. If true, you agree to the Let's Encrypt terms of service (default false)
  -o, --output string                         Output manifests instead of applying them (json, yaml)
      --pod-security-enforce-profile string   The name of the PodSecurity profile to set (default baseline)
      --set-pod-security-enforce-profile      Set the PodSecurity profile on created namespaces (default true)
      --skip-checks                           Bypass installation checks
```

### Options inherited from parent commands

```
  -A, --all-namespaces      Namespace to work in
      --context string      Context to use in the kubeconfig file
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Location of a kubeconfig file
      --namespace string    Namespace to work in (default "acorn")
```

### SEE ALSO

* [acorn](acorn.md)	 - 

