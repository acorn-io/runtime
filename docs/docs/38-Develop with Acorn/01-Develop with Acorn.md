---
title: Develop with Acorn
---

## Already have an Acorn file setup

To start developing with Acorn, from the directory with the Acorn file run:

`acorn dev .`

This will start all of the services, expose all of the ports, and stream output to the command line.

Files will be synchronized from the local file system into the container. Apps that support live reloading when they detect file changes will be able to take advantage of the syncing.

## Setting up Acorn for development

Acorn provides a dev mode that deploys an environment onto a cluster and provides file syncing to the remote containers. This allows application developers to have a single source of truth to describe the development and production deployments.

To develop a Hugo app, download and install Hugo.

Initialize a new site

```shell
> hugo new site acorn-demo
> cd acorn-demo
> git init .
> git submodule add https://github.com/theNewDynamic/gohugo-theme-ananke.git themes/ananke
```

### Dockerfile

Create a Dockerfile with the following:

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

There are two targets defined in this Dockerfile, one for publishing to production and the other for development.

### Acorn file

Next an Acorn file is needed to define the development and production behavior. to make edits with live reload and to publish. This example is making use of the built in `args.dev` boolean value. It is set to true when running `acorn dev .`. If you are using the `acorn dev render .` command you will need to pass the profile flag: `acorn dev --profile dev render .`

```cue
containers: {
    app: {
        build: {
            context: "."
        }

        if args.dev {
            build: target: "dev"
            expose: "1313/http"
        }
    
        if !args.dev {
            build: target: "prod"
            expose: "80/http"
        }
    }
}
```

The above Acorn file will expose port 1313 in development mode to match the Dockerfile build target "dev". In our "prod" build target, we are running in an nginx container that exposes port 80.

#### Setup file syncing

In order to sync files from our development machine to the running container, we need to specify a directory in our "dev" block.

```cue
containers: {
    app: {
        build: {
            context: "."
        }

        if args.dev {
            build: target: "dev"
            expose: "1313/http"
            dirs: {
                "/src": "./"
            }
        }
    
        if !args.dev {
            build: target: "prod"
            expose: "80/http"
        }
    }
}
```

Now when `acorn dev` is run against this file it will build the `dev` target for this image. It will sync the local directory to the `/src` folder in the container.

Now when content is added to the site, it is automatically rendered so you can view it locally. When ready to publish, do an `acorn build .` The static content will be generated and packaged in the `nginx` container and deployable as an Acorn.

## Shipping the Acorn

Once development has reached a point where it needs to be deployed to production, it needs to be built.

Build the Acorn image:
`acorn build -t registry.example.com/myorg/my-app:latest .`

Once built, push to a registry

`acorn push registry.example.com/myorg/my-app:latest`

This Acorn can now be run like:

`acorn run -d my.blog.example.com:app registry.example.com/myorg/my-app:latest`

This will deploy the Acorn and expose it on `my.blog.example.com`.
