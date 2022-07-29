# GitHub Actions

## Quick Start

Use a GitHub [Action](https://github.com/features/actions) to build and push a new Acorn into a [Packages](https://ghcr.io/) repository each time a new version is tagged:

```yaml
# Save into your repo as .github/workflows/on-tag.yaml
name: On Tag
on:
  push:
    tags:
    - "v*"
jobs:
  publish:
    steps:
      - uses: actions/checkout@v3
      - uses: acorn-io/actions-setup@v1
      - uses: acorn-io/actions-login@v1
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Set Tag
        run: |
          echo "TAG=${GITHUB_REF#refs/*/}" >> $GITHUB_ENV
      - name: Build and Push
        run: |
          acorn build -t ghcr.io/${{ github.repository }}:$TAG .
          acorn push ghcr.io/${{ github.repository }}:$TAG
```

The action is automatically provided with a `GITHUB_TOKEN` that already has permission to push to the registry for its own repo, so no special secrets configuration is needed.

## Setup Action

The setup action creates a k3s cluster, installs acorn, and hooks everything up so you can use the `acorn` CLI just like you would from your workstation.

### Usage
```yaml
name: My Workflow
on:
  push: {}
jobs:
  publish:
    steps:
      - uses: actions/checkout@v3
      - uses: acorn-io/actions-setup@v1
      - run: |
        # Do something with the CLI
        acorn --version
```

### Options

| Key             | Default  | Description |
| --------------- | ---------| ----------- |
| `acorn-version` | `latest` | Version of Acorn to install
| `k3s-version`   | `latest` | Version of K3s to install

See [actions-setup](https://github.com/acorn-io/actions-setup#readme) for additional advanced options.  For example it is possible to point acorn at an existing k8s cluster instead of spinning up a new one.

## Login Action

The login action logs into a registry so that later steps can push an acorn to it.

### Usage

```yaml
name: My Workflow
on:
  push:
    tags:
    - "v*"
jobs:
  publish:
    steps:
      - uses: actions/checkout@v3
      - uses: acorn-io/actions-setup@v1
      - uses: acorn-io/actions-login@v1
        with:
          registry: docker.io
          username: yourDockerHubUsername
          password: ${{ secrets.DOCKERHUB_PASSWORD }}
```

### Options

| Key        | Default      | Description |
| ---------- | ------------ | ----------- |
| `registry` | **Required** | Registry address to login to (e.g. `ghcr.io` or `docker.io`)
| `username` | **Required** | Registry username
| `password` | **Required** | Registry password
