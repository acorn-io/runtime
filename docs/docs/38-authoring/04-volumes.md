---
title: Volumes
---

Volumes are used to store persistent data in your applications. Volumes can be bound to containers, sidecars, and jobs whenever persistence is needed. Defining a volume in the Acornfile is done under the `volumes` key and referenced via the `volumes://` uri path when mounting.

```acorn
containers: {
    "my-app": {
        image: "web"
        // ...
        dirs: {
            "/data": "volume://my-data"
        }
    }
}
// ...
volumes: {
    "my-data": {}
}
// ...
```

In the above example, there is a `my-data` volume defined and mounted into the `my-app` container at the `/data` path. The volume will create a 10G volume using the `default` storage class defined in the cluster. The default volume type will be created as a `ReadWriteOnce` volume and consumable by multiple containers on a single host.

A volume has the following fields that can be customized, here is the above volume defined with all of the fields.

```acorn
volumes: {
    "my-data": {
        size: 10G
        class: "default"
        accessModes: "readWriteOnce"
    }
}
```

## Volumes with sidecars

Sidecars can share volumes with the primary app container or have volumes for their exclusive use. In order to share data, a volume must be created and mounted in both containers.

```acorn
containers: {
    frontend: {
        image: "nginx"
        dirs: {
            "/var/www/html": "volume://web-content"
        }
        // ...
        sidecars: {
            image: "git-cloner"
            // ...
            dirs: {
                "/var/www/html": "volume://web-content"
            }
        }
    }
}
// ...
volumes: {
    "web-content": {}
}
```

In the above example both containers will have read-write access to the data in `volume://web-content`.

A volume can also be exclusively mounted in a sidecar container.

## Ephemeral storage

There are two ways to create ephemeral scratch type of storage. This type of volume is useful when you are transforming data perhaps during a restore process.

A shorthand way to define the volume is:

  ```acorn
containers: {
    frontend: {
        // ...
        dirs: {
            "/scratch": "ephemeral://scratch-data"
        }
    }
}
```

The above is equivalent to:

```acorn
containers: {
    frontend: {
        // ...
        dirs: {
            "/scratch": "volume://scratch-data"
        }
    }
}

volumes: {
    "scratch-data": {
        class: "ephemeral" 
    }
}
```

The `ephemeral` class is a special case that Acorn will handle behind the scenes to create an `emptyDir` volume.

## Volumes with jobs

Volumes can also be mounted between app containers and job containers.

```acorn
containers: {
    db: {
        // ...
        dirs: {
            "/var/lib/db_data": "volume://db-data"
        }
        // ...
    }
}

volumes: {
    "db-data": {}
    "backups": {}
}

jobs: {
    backups: {
        // ...
        dirs: {
            "/backups": "volume://backups"
            "/var/lib/db_data": "volume://db-data"
        }
        // ...
    }
}
```
