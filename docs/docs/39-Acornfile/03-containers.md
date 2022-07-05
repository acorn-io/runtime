---
title: Containers
---

## Defining a container

The container definition in the Acorn file defines everything about the individual components needed to run your app. Each container definition will define a service in your application Acorn. A basic example of a container definition that exposes a network endpoint looks like the following:

```cue
containers:{
    "my-webapp": {
        image: "nginx"
        expose: "80/http"
        env: {
            "NGINX_HOST": "example.com"
        }
    }
}
```

The above file defines a container that will be deployed named "my-webapp" and it will expose port 80 to the cluster it is running on, and publicly if needed at runtime. There is also an environment variable `NGINX_HOST` being set to `example.com`.

If the application exposes more then one port, the expose field can be a list like so:

```cue
containers:{
    "my-webapp": {
        image: "nginx"
        expose: [
          "80/http",
          "443/tcp",
        ]
        env: {
            "NGINX_HOST": "example.com"
        }
    }
}
```

```cue
containers:{
    "my-webapp": {
        image: "nginx"
        expose: [
          "80/http",
          "443/tcp",
        ]
        env: {
            "NGINX_HOST": "example.com"
        }
    }
    database: {
        image: "mysql"
        ports: "3306/tcp"
    }
}
```

This Acorn file defines two containers, one called `my-webapp` and the other `database`. In the database container, we are using the `ports` key. This means that only other containers running in this Acorn, in this case `my-webapp` can access the port.

Also, in the above examples there is an environment variable being defined to

## Defining a container from a Dockerfile

Acorn provides a mechanism to build your own containers for your application. If you have an existing project that already defines an Dockerfile, you can build it from Acorn.

```cue
containers: {
    "my-app": {
        build: {
            context: "."
        }
        expose: "3000/http"
    }
}
```

Now when an `acorn build .` or `acorn dev .` is run, the `my-app` container will be built and packaged as a part of the Acorn. It will look for a Dockerfile in the `./` directory. You can specify a different file or location with the dockerfile key.

```cue
containers: {
    "my-app": {
        build: {
            context: "."
            dockerfile: "./pkg/Dockerfile.prod"
        }
        expose: "3000/http"
    }
}
```

## Defining sidecar containers

Sometimes a container needs some setup before it runs, or has additional services running along side it. For these scenarios, the `sidecar` can be defined as part of the container.

```cue
containers: {
    frontend: {
        image: "nginx"
        ...
        sidecars: {
            "git-clone": {
                image: "my-org/git-cloner"
                init: true
            }
            "metrics-collector": {
                image: "my-org/metrics-collector"
                ports: "5000/http"
            }
        }
    }
}
```

In the above file, we have a two sidecars defined. One is `git-clone` which is defined as an `init` container. The init container starts up before the primary container. Each init container should run a single task, and must complete successfully before additional init and application containers are started.

The second side car above is a service that runs alongside the primary frontend container and in this case provides a metrics endpoint. You can define as many side car containers as you need to run and support your application.

## Additional Reading

* [Networking Concepts in Acorn](/concepts/networking)
* [Acorn file reference](/reference/acorn.cue)
