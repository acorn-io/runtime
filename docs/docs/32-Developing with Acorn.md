---
title: Developing Apps with Acorn
---

## Acorn dev

### Already have an Acorn file setup

To start developing with Acorn, from the directory with the Acorn file run:

`acorn dev .`

This will start all of the services, expose all of the ports, and stream output to the command line.

Files will be synchronized from the local file system into the container. Apps that support live reloading when they detect file changes will be able to take advantage of the syncing.

### Setting up Acorn for development

Acorn provides a dev mode that deploys an environment onto a cluster and provides file syncing to the remote containers. This allows application developers to have a single source of truth to describe the development and production deployments.

To develop a Hugo app, download and install Hugo.

Initialize a new site

```shell
> hugo new site acorn-demo
> cd acorn-demo
> git init .
> git submodule add https://github.com/theNewDynamic/gohugo-theme-ananke.git themes/ananke
```

Create a Dockerfile with the following:

```dockerfile
FROM klakegg/hugo:0.101.0-alpine AS hugo

ADD . /src
WORKDIR /src
RUN mkdir -p /target && \
    hugo -d /target/ --minify

FROM nginx AS static
COPY --from=hugo /target /usr/share/nginx/html

FROM hugo AS dev
EXPOSE 1313
CMD [ "server", "--bind", "0.0.0.0", "-D" ]
```

There are two targets defined in this Dockerfile, one for publishing to production and the other for development.

Acorn file to make edits with live reload and to publish.

```cue
containers: {
    app: {
        build: {
            context: "."
        }

        if args.dev {
            build: target: "dev"
            expose: "1313/http"
            dirs: "/src": "./"
        }
    
        if !args.dev {
            build: target: "static"
            expose: "80/http"
        }
    }
}
```

Now when `acorn dev` is run against this file it will build the `dev` target for this image. It will sync the local directory to the `/src` folder in the container.

Now when content is added to the site, it is automatically rendered so you can view it locally. When ready to publish, do an `acorn build .` The static content will be generated and packaged in the `nginx` container and deployable as an Acorn.
