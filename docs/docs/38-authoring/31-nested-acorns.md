---
title: Nested Acorns
---

Nested acorns allow you to describe a collection of microservices that make up a complete deployment of an Application. This is useful when describing entire deployments along with profiles for their arguments. Combined with the auto-upgrade functionality, it allows you to create a delivery pipeline in a single file that can be checked into VCS. Additionally, Acorns can also be used within an Acorn app to provide functionality to the app. The use of Acorns can be thought of as using a module or library in a programming language.

## Using nested Acorn

Acorns are defined under the `acorns` toplevel key in the Acornfile.

```acorn
//...
acorns: {
    "my-acorn": {
        image: "ghcr.io/acorn-io/hello-world:latest"
    }
}
//...
```

## Describe a deployment pipeline with Acorns and services

To describe an entire deployment with Acorns you declare each Acorn and define a profile for the deployment type. You can also use `services` to provide infrastructure components to the app deployment.

```acorn
args: {
    uri: ""
    tag: ""
}

profiles: {
    dev: {
        uri: "index.io/my-app/dev/secret-name"
        tag: "latest"
    }
    prod: {
        uri: "index.io/my-app/prod/secret-name"
        tag: "1.2.#"
    }
}

acorns: {
    "my-app": {
        image: "example.com/repo/org/hello-world:\(args.tag)"
        secrets: ["secrets-getter.this:redis-creds"]
        autoUpgrade: true
    }

}

services: {
    // A service that returns a secret from an external source under the name `this`
    "secret-getter": {
        image: "example.com/repo/org/secret-getter:latest"
        serviceArgs: {
            secretURI: args.uri
        }
    }
}
```

Now when this Acornfile is initially deployed with the `--profile` flag Acorn will deploy the app with the appropriate default values. In this case `dev` will always update whenever the latest tag moves. Production will always update on new patch releases.
