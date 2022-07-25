---
title: Multiple Containers
---

## Multiple containers

A lot of applications are made up of multiple components which often run in different containers. A web framework might be making use of a database instance.

Acorn allows multiple containers to be defined in the same Acornfile, and provides some convenient mechanisms for connecting them.

### Adding a second container

Building on our previous Acornfile, lets add a database container:

```cue
containers: {
  app: {
    build: {
      context: "."
    }
  }
  mariadb: {
    image: "mariadb"
    env: {
        "MARIADB_ROOT_PASSWORD": "password"
        "MARIADB_USER":          "app-user"
        "MARIADB_PASSWORD":      "password"
        "MARIADB_DATABASE":      "app-db"
    }
  }
}
```

Now when Acorn builds this, it will build the `app` container from the Dockerfile and pull the `mariadb` image from Dockerhub. When run there will be two containers associated to this app. The containers will be able to communicate with each other through the container names `mariadb` and `app`.

### Env vars and secrets

In the last example we added the `env` struct to the `mariadb` container to set some variables needed for the container to start, a full list can be found on the image's Dockerhub page. The `env` struct is a set of key value pairs that will be passed into the container.

The example above is setting passwords as hard coded plain text strings, which isn't typically what is needed for an application. Here we can create a closed loop system of auto-generated secrets that no person needs to know ahead of time.

Edit the Acornfile to add some secrets for the password.

```cue
containers: {
  app: {
    build: {
      context: "."
    }
    env: {
        "DB_NAME": "app-db"
        "DB_USER": "secret://mariadb-app-db-user-creds/username"
        "DB_PASS": "secret://mariadb-app-db-user-creds/password"
    }
  }
  mariadb: {
    image: "mariadb"
    env: {
        "MARIADB_ROOT_PASSWORD": "secret://mariadb-root-password/token"
        "MARIADB_USER":          "secret://mariadb-app-db-user-creds/username"
        "MARIADB_PASSWORD":      "secret://mariadb-app-db-user-creds/password"
        "MARIADB_DATABASE":      "app-db"
    }
  }
}
secrets: {
    "mariadb-root-password": {
        type: "token"
    }
    "mariadb-app-db-user-creds": {
        type: "basic"
    }
}
```

In the above example, we are generating two secrets. One is for the root password of the `mariadb` container and the other is the db user credentials that will be used in both containers. It assumes that the app will startup and be able to find it's database credentials by looking at those environment variables.

This is helpful in development and deployments, because the app will be able to use unique credentials that do not need to be known ahead of time.
