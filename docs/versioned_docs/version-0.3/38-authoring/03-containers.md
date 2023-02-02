---
title: Containers
---

## Defining a container

The container section in the Acornfile defines everything about the individual components needed to run your app. Each container definition will define a service in your Acorn application. A basic example of a container definition that exposes a network endpoint looks like the following:

```acorn
containers:{
    "my-webapp": {
        image: "nginx"
        ports: publish: "80/http"
    }
}
```

## Defining a container built from a Dockerfile

### Standard build options

Acorn provides a mechanism to build your own containers for your application. If you have an existing project that already defines an Dockerfile, you can build it using Acorn.

```acorn
containers: {
    "my-app": {
        build: {
            context: "."
        }
    }
}
```

Now when `acorn build .` or `acorn run -i .` is run, the `my-app` container will be built and packaged as a part of the Acorn. It will look for a Dockerfile in the `./` directory.

### Customized build behavior

Acorn provides options to customize the building of OCI images. If the Dockerfile is not in the root directory of the `context` you can specify the location using the `dockerfile` parameter. If the image is a multi-stage build, the desired target can be specified. See [args and profiles](07-args-and-profiles.md) to see how to customize these values at build and runtime.

```acorn
containers: {
    "my-app": {
        build: {
            context: "."
            dockerfile: "./pkg/Dockerfile.prod"
            target: "dev"
        }
    }
}
```

You can also specify a `buildArgs` object as part of the build section to pass arguments to the build.

```acorn
containers: {
    app: {
        build: {
            // ...
            buildArgs: {
                param: "value"
                // ...
            }
        }
    }
}
```

## Network ports

### Basic definition

Most containers can be interacted with over the network. In order for containers to be reached by other containers and applications, the network ports must be defined as part of the container.

A port definition is:

```shell
[NAME]:<PORT>[/TYPE]
```

Where:

* `NAME`: is an optional name for the port, like `db` or `metrics`.
* `PORT`: is required numeric value of the port.
* `TYPE`: is an optional value of `UDP`, `TCP`, or `HTTP`.

An example of a named port:

```acorn
containers: {
    db: {
        image: "mysql"
        ports: expose: "mysql:3306/tcp"
    }
}
```

When the user runs the built Acorn image, they can reference this port by name `--expose sql:3306:mysql:3306`.

### Scopes

As an author, there are three scopes used to define the ports **default** access behavior `internal`, `expose`, and `publish`. These settings can be changed at runtime by the operator.

| Scope | Accessibility |
| ------| --------------|
| `internal`| Containers within the Acorn image (default)|
| `expose` | Containers running across the cluster |
| `publish` | Accessible outside the cluster by consumers |

```acorn
containers: {
    "my-webapp": {
        image: "nginx"
        ports: [
            "5000/http",
        ]
    }
}
```

The above example defines an HTTP port `5000` accessible to other containers defined in this Acorn image.

The next example shows the `expose` and `publish` settings, used to define ports that are meant to be accessed outside of the Acorn app or published outside the cluster. If an Acorn image is going to provide a network service to multiple apps, something like a database, you will likely want to expose the port so it can be linked to other Acorn apps.

```acorn
containers: {
    "my-webapp": {
        image: "nginx"
        ports: {
            expose: "5001/http"
            publish: "8080/http"
        }
    }
    database: {
        image: "mysql"
        ports: "3306/tcp"
    }
}
```

This Acornfile defines two containers, one called `my-webapp` and the other `database`. The `my-webapp` container is exposing port 8080 outside of the cluster and port 5001 to apps running on the cluster. When launching this Acorn the port can be published outside the cluster or accessed by linked Acorns.

In the database container, we are using the `ports` parameter because only the my-webapp container will communicate with the database.

## Environment variables

Containers often use environment variables as a way to configure common settings or pass in secrets. Acorn supports both methods in the container definition.

```acorn
containers: {
    db: {
        image: "mysql"
        // ...
        env: {
            "MYSQL_ROOT_PASSWORD": "secret://root-pass/token"
            "DATABASE_NAME": "test-app"
            "USER_SET_VALUE": args.userValue
        }
    }
}
```

The above example has a `db` container with the `MYSQL_ROOT_PASSWORD` variable set by a [secret](05-secrets.md) in the Acornfile. The `DATABASE_NAME` is set to a static value, and the `USER_SET_VALUE` is defined by a user [arg](07-args-and-profiles.md). When launched the container can access these environment variables as needed.

## Files

Files are defined in a container where the key is the location inside the container, and the value is the contents of the file.

```acorn
containers: {
    web: {
        image: "nginx"
        // ...
        files: {
            "/etc/htpasswd": "secret://htpasswd-file/content"
            "/usr/share/nginx/html/index.html": "<h1>Hello!</h1>"
        }
        // ...
    }
}
```

In the above, the file `/etc/htpasswd` will contain the contents from a secret that contains the appropriate values for an htpasswd file. The `/usr/share/nginx/html/index.html` will contain static content from the string.

## Directories

Directories are defined in their own block and are used to mount volumes, files from a secret, or
pull in content at build time. The block has keys that are path names inside the container and values are the source.

```acorn
containers: {
    web: {
        image: "nginx"
        // ...
        dirs:{
            "/init-scripts": "./scripts"
            "/home/.ssh/": "secret://ssh-keys"
            "/data": "volume://data-vol"
        }
        // ...
    }
}
```

In the above file, when building the Acorn the `/init-scripts` directory will be populated with the contents of the local `./scripts` directory. Files are copied in with the same permissions.

The `/home/.ssh/` directory will have files named after the secrets keys and content. The `/data` directory will have the volume `data-vol` mounted.

## Probes

Applications running for a long time sometimes fail in strange ways, or take time to startup before they are ready to receive traffic. To ensure the container is running and ready to receive traffic there are probes.

There are three types of probes: `readiness`, `liveness`, and `startup`. Probes are defined per container in a list. You can define one of each type per container. If the probes fail, the container is restarted.

Each probe type has the following parameters that can be optionally set.

| Parameter | Default | Description |
| ----------|---------|-------------|
| `initialDelaySeconds`| 0 |  Number of seconds to wait after the container is started before probes are initiated. |
| `periodSeconds` | 10 | Number of seconds between probe attempts. |
| `timeoutSeconds` | 1 | Number of seconds before the probe times out. |
| `successThreshold` | 1 | Number of consecutive successful probes before considering the container healthy. |
| `failureThreshold` | 3 | Number of consecutive failed probes before considering the container unhealthy. |

There are three types of checks that can be used to check the health of the container. A script can be executed inside the container, an HTTP endpoint can be checked, or a TCP endpoint can be checked. Each of the probe types can use one of any of these check types.

### Liveness probes

The liveness probe detects that the container is up and considered running. The example here defines an HTTP type of check, though TCP and exec types could also be used.

```acorn
containers: {
    web: {
        image: "nginx"
        // ...
        probes: [
            {
                type: "liveness"
                http: {
                    url: "http://localhost/healthz"
                    headers: {
                        Accept: "application/json"
                    }
                }
            },
        ]
    }
}
```

Headers are an optional field on the HTTP health probe type.

### Readiness probes

A readiness probe means the container is ready to receive traffic. Sometimes when a database server starts it needs to have data loaded. During this data loading process, it should not be contacted by the application in case it has incomplete data. You can use the readinessprobe probe to prevent other containers from accessing this container before it is ready.

This example will use an exec check, but HTTP and TCP checks could also be used.

*Note: Readiness probes do not wait for the liveness probe to succeed first. If you need the readiness probe to wait you should use the `initialDelaySeconds` parameter to delay. Alternatively, the startup probe can also be used.*

```acorn
containers: {
    db: {
        image: "mysql"
        env: {
            MYSQL_ROOT_PASSWORD: "secret://db-root-pass/token"
            // ...
        }
        // ...
        probes: [
            {
                type: "readiness"
                initialDelaySeconds: 10
                exec: command: [
                    "mysqladmin",
                    "status",
                    "-uroot",
                    "-p${MYSQL_ROOT_PASSWORD}",
                ]
            },
        ]
    }
}
```

### Startup probes

Startup probes exist to give slow starting applications time to load data and/or configuration before starting the liveness and readiness probes. The startup probe should use the same command, HTTP, or TCP check as the liveness probe with enough time to cover the worst case startup scenario. The time is calculated by `failureThreshold * periodSeconds`.

```acorn
containers: {
    web: {
        image: "nginx"
        // ...
        probes: [
            {
                type: "liveness"
                tcp:{
                    url: tcp://localhost:80
                }
            },
            {
                "type: "startup"
                failureThreshold: 10
                periodSeconds: 6
                tcp:{
                    url: tcp://localhost:80
                }
            }
        ]
    }
}
```

In the above example the `web` container would have 60 seconds (10 tries * 6 seconds) to startup before being restarted.

## Defining sidecar containers

Sometimes a container needs some setup before it runs, or has additional services running along side it. For these scenarios, the `sidecar` can be defined as part of the container.

```acorn
containers: {
    frontend: {
        image: "nginx"
        // ...
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

In the above file, we have two sidecars defined. One is `git-clone` which is defined as an `init` container. The init container starts up before the primary container. Each init container should run a single task, and must complete successfully before additional init and application containers are started.

The second sidecar above is a service that runs alongside the primary frontend container and in this case provides a metrics endpoint. You can define as many sidecar containers as you need to run and support your application.

## Additional Reading

* [Acorn Security Considerations](60-architecture/02-security-considerations.md)
* [Acornfile reference](100-reference/03-acornfile.md)
