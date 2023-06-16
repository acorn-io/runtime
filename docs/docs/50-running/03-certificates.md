---
title: TLS Certificates
---

Applications that publish HTTP endpoints can be protected by TLS certificates.  If you've enabled Acorn's [Let's Encrypt integration](30-installation/02-options.md#tls-via-lets-encrypt), a valid certificate will be provisioned for your app's endpoints. This only applies to oss-acorn.io generated endpoints. For custom endpoints configured using the [publish flag](50-running/02-networking.md#publish-individual-ports), acorn relies on external cert-manager to issue certificates. For more information on how to configure cert-manager, see [Issuing custom domain certs](#issuing-custom-domain-certs).

## Manually adding certificates
If you don't wish to use Acorn's Let's Encrypt integration, you can configure certificates manually or by integrating with cert-manager. Acorn will automatically look for SANs in secrets of type `kubernetes.io/tls` for the exposed FQDN of the application in the Acorn namespace.

The following examples assume you are deploying an app and plan to host on `my-app.example.com`

### Add existing certificates using kubectl

Before launching the application pre-create a secret in the `acorn` namespace containing the
certificate like so:

`kubectl create secret tls my-app-tls-secret --cert=path/to/my-app-tls.cert --key=path/to/my-app-tls.key`

### Add with Cert-Manager

If you are already using Cert-Manager today, you can leverage it with Acorn right away. First you must
create a certificate resource in the Acorn namespace:

`kubectl apply -n acorn -f ./my-cert.yaml`

```yaml
# my-cert.yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: registry-studio-cert
spec:
  dnsNames:
  - my-app.example.com
  issuerRef:
    group: cert-manager.io
    kind: ClusterIssuer
    name: prod-issuer
  secretName: my-app-tls-secret
```

Cert-Manager will create a certificate for `my-app.example.com` and store it in a secret `my-app-tls-secret`.

### Consume the secret

Once you have manually created the TLS secret using one of the methods above you can consume it in your application.

When you deploy the application Acorn, you can launch with the FQDN of your app.

```shell
acorn run -p my-app.example.com:web [MY_APP_IMAGE]
```

Acorn will automatically inspect each certificate in the Acorn namespace for one that can be used with `my-app.example.com`.
If no TLS secret is found with that FQDN, it will be exposed on HTTP only.

### Issuing custom domain certs

Acorn Let's Encrypt integration does not issue certificates for custom domain. Instead, you will rely on external cert-manager to issue certificates. To do so, you will need to create a cluster issuer first. For more information on how to configure cert-manager, see [cert-manager docs](https://cert-manager.io/docs/).

```yaml 
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: users@exmaple.io
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
      - http01:
          ingress:
            ingressClassName: traefik
```

Modify the ingressClassName to match the ingress controller you are using. For example, if you are using nginx ingress controller, you will need to change it to `nginx`.

Once you have created the cluster issuer, pass the cluster issuer's name to acorn install so that acorn knows where to apply the cluster issuer to the ingress resource.

```shell
acorn install --cert-manager-issuer=letsencrypt-prod
```