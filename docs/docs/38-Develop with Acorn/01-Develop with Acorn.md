---
title: Developing with Acorn
---

Developing applications often requires configurations different from the target production setup. Different frameworks have development servers that listen on unique ports, the scale of systems might be lower due to resource constraints, etc that require runtime setups. This section will cover how to address configuration differences between development and production.

## Setup for Development mode

You can checkout the [source code](https://github.com/acorn-io/docs-examples) for this example on GitHub. The complete Acornfile is in the repository, or you can follow along this guide to build the file. In the repository you will see a Dockerfile that looks like this:

```dockerfile
FROM klakegg/hugo:0.101.0-alpine AS hugo

ADD . /src
WORKDIR /src
RUN mkdir -p /target && \
    hugo -d /target/ --minify

FROM nginx AS prod
COPY --from=hugo /target /usr/share/nginx/html

FROM hugo AS dev
EXPOSE 1313
CMD [ "server", "--bind", "0.0.0.0", "-D" ]
```

## Acornfile

First, an Acornfile is needed to define the development and production behavior. In production the Hugo app will run in an Nginx container on standard port 80, but for development Hugo provides a dev server that runs on port 1313.

To handle this difference of port configuration and Docker build targets, the example will make use of the built-in `args.dev` boolean value. It is set to true when running `acorn run -i .`.

Create the Acornfile with the following content.

```cue
containers: {
    app: {
        build: {
            context: "."
        }

        if args.dev {
            build: target: "dev"
            ports: publish: "1313/http"
        }
    
        if !args.dev {
            build: target: "prod"
            ports: publish: "80/http"
        }
    }
}
```

The above Acornfile will build the Docker image to the 'dev' target and expose port 1313. When building an Acorn for production it will build the "prod" build target and expose port 80.

If this is run via `acorn run -i .`, it will expose port 1313 on an endpoint that you can see on the development machine.

## Acorn live edit mode

Most teams will want to develop their app, and update when source code changes. To accomplish this, Acorn can be configured to synchronize files from the local filesystem into the container. Apps that support live reloading or hot-reloading when they detect file changes will be able to take advantage of the syncing.

### File syncing

In order to sync files from your development machine to the running container, we need to specify a directory in our "dev" block.

```cue
containers: {
    app: {
        build: {
            context: "."
        }

        if args.dev {
            build: target: "dev"
            ports: publish: "1313/http"
            dirs: {
                "/src": "./"
            }
        }
    
        if !args.dev {
            build: target: "prod"
            ports: publish: "80/http"
        }
    }
}
```

Now when `acorn run -i .` is run against this file it will build the `dev` target for this image, run it and sync the local directory to the `/src` folder in the container.

Now when content is added to the site, it is automatically rendered so you can view it locally. When ready to publish, do an `acorn build .`. The static content will be generated and packaged in the `nginx` container and is then deployable as an Acorn.

## Shipping the Acorn

Once development has reached a point where it needs to be deployed to production, it needs to be built.

Build the Acorn image:
`acorn build -t registry.example.com/myorg/my-app:latest .`

Once built, push it to a registry

`acorn push registry.example.com/myorg/my-app:latest`

This Acorn can now be run via

`acorn run -p my.blog.example.com:app registry.example.com/myorg/my-app:latest`

This will deploy the Acorn and expose it on `my.blog.example.com`.
