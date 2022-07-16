---
title: Persisting data
---

## Volumes

With the multi-container example, you might have observed the database is wiped every time the container restarts. Which might be tolerable in development, but definitely not for production. To ensure the database survives container restarts a volume should be created and attached.

### Adding a volume to a container

Building on the multi-container Acornfile lets define a volume:

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
    dirs: {
        "/var/lib/mysql": "volume://db-data"
    }
  }
}
volumes: {
    "db-data": {}
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

The above example defines a volume `db-data` and mounts it into the container at `/var/lib/mysql`. This volume will persist container restarts and updates to the Acorn file.
