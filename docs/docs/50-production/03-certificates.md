---
title: TLS Certificates
---

Services exposing HTTP endpoints can be protected by TLS certificates. In the
future, Acorn will provide built-in mechanisms to automatically provide certificates
for each endpoint. Today, adding a certificate must follow the manual approach.

<!-- TODO: ## Automatic -->

## Manually adding certificates

Acorn will automatically look for SANs in secrets of type `kubernetes.io/tls` for the
exposed FQDN of the application in the Acorn namespace.

Assume you are deploying an app and plan to host on `my-app.example.com`

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
