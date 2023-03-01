---
title: Acorn Linkerd Plugin
---

:::caution
This is an EXPERIMENTAL feature.
:::

Acorn Labs provides a [plugin](https://github.com/acorn-io/acorn-linkerd-plugin) for Acorn that integrates with the
[Linkerd service mesh](https://linkerd.io) to accomplish the following:

1. Set up Linkerd sidecar injection for all deployed Acorn apps.
2. Create Linkerd policies to block cross-project network traffic.

## Installation

### Prerequisites

Acorn and Linkerd must already be installed in the Kubernetes cluster. Acorn must be configured to propagate the
following project annotations:

- `linkerd.io/inject`
- `config.linkerd.io/default-inbound-policy`

Additionally, unless the cluster is using the [Linkerd CNI](https://linkerd.io/2.12/features/cni/), Acorn must be
configured to disable PodSecurity enforcement, since the initContainers created by Linkerd require more privileges than
the baseline PodSecurity allows.

This is an example of how to install Linkerd and Acorn to fulfill these requirements:

```bash
linkerd install --crds | kubectl apply -f -
linkerd install | kubectl apply -f -
acorn install --propagate-project-annotation "linkerd.io/inject","config.linkerd.io/default-inbound-policy" --set-pod-security-enforce-profile=false
```

For more information, see the [`acorn install` reference page](../100-reference/01-command-line/acorn_install.md).

### Installing the Plugin

The plugin is provided as an acorn and can be installed with `acorn run`:

```bash
acorn run --name acorn-linkerd-plugin ghcr.io/acorn-io/acorn-linkerd-plugin:main
```

The plugin requires many permissions to interact with resources in the cluster, so Acorn will ask for confirmation
before running it.

Any new Acorn apps deployed after the plugin is installed will be automatically brought into the service mesh.
Apps that were already running before the plugin was installed can be meshed by restarting them:

```bash
acorn stop <app name>
acorn start <app name>
```

## Known Issues and Limitations

### Readiness Probes

The readiness probes that Acorn creates for apps that specify ports in their Acornfile are not compatible with Linkerd.
They will appear to always succeed (so the application will always appear to be ready), because the proxy will
receive the traffic and always respond, even if the application itself is not running or isn't ready.

#### Workaround

This problem can be avoided by specifying an HTTP or exec probe in the Acornfile, as those will still function
properly with Linkerd.

:::caution
HTTP probes will cause Linkerd to stop enforcing policies on the specified port. For example, if `http://localhost:80`
is specified as the readiness probe in the Acornfile of an app, all incoming traffic will be allowed to reach the app
on port 80, including traffic **from other Acorn projects**.
:::

### Cross-Project Pings

ICMP ping traffic cannot be blocked by Linkerd because it does not use TCP or UDP. As a result of this, it is still
possible for apps to ping each other across separate Acorn projects even with the plugin installed.
