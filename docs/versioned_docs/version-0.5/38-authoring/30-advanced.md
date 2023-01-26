---
title: Advanced Topics
---

## Scaling Applications

### Stateless applications

In the case of scaling stateless applications the `scale` field can be defined on the container to increase the number of containers deployed. This is where consumers of the Acorn won't need persistent data or to know which instance will be deleted on scale down operation.

```acorn
args: {
    // Number of stateless web servers to run
    scale: 1
}

containers: {
    web: {
        image: "web"
        scale: args.scale
    }
}
```

### Stateful applications

Applications that have stateful data or where an operator would care in which order the containers will be removed in a scale down event should not use the `scale` field and should instead create unique instances of the container.

To accomplish this, users can leverage `for` loops in the Acornfile. Within the `for` loop all items unique to that instance should be defined. In most cases, this will be a container and data volumes. The loop can contain any of the top level objects if needed.

```acorn
args: {
    // Number of instances
    replicas: 1
}

for i in std.range(0, replicas, 1) {
    containers: {
        "instance-\(i)": {
            // ...
            dirs: {
                "/data": "volume://instance-data-\(i)"
            }
        }
    }
    volumes: {
        "instance-data-\(i)": {}
    }
}
```

The above example makes use of the `std.range` function used in the `for` loop. The loop variable `i` will be an integer and placed into the container and volume names. When the application is scaled up, new containers will be deployed with their own data volumes. When the application is scaled down the highest numbered replicas will be removed first. The `0` replica will always be the first replica deployed and last removed.

When deploying stateful applications it is a reasonable assumption to bootstrap from the `0` instance and for new replicas to use that as the first point of contact to register.

### Yaml templates for config files

If you would like to dump a section of the localData config into YAML format, you can use the YAML encoder package.

```acorn
args: {
    // User provided yaml
    userConfig: {}
}

containers: {
    frontend: {
        // ...
        files: {
            "/my/app/config.yaml": "secret://yaml-config/template"
        }
        // ...
    }
}

secrets: {
    "yaml-config": {
        type: "template"
        data: {
            template: std.toYAML(localData.config)
        }
    }
}

localData: {
    config: std.merge({
        this: {
            isGoing: {
                to: "be a yaml file"
            }
        }}, args.userConfig)
}
```

In the above example the frontend config file will be rendered from user and Acorn data in YAML format. This example is using the `std.merge()` function which takes two objects and merges them where the second overwrites the first.

### Generating files from key value pairs

Another useful built-in for rendering key value pairs with an optional separator is the `std.join` function.

If you need to create a file with content in this format:

`key=value`

```acorn
// ...
containers: {
    web: {
        // ...
        files: {
            "/etc/config_file": "secret://config/template"
        }
        // ...
    }
}
secrets: {
    "config": {
        type: "template"
        data: {
            template: std.join([for key, value in localData.configData {"\(key)=\(value)"}], "\n")
        }
    }
}
localData: {
    configData: {
        key: "value1"
        key0: "value2"
    }
}
```

The above will output into /etc/config_file:

```ini
key=value1
key0=value2
```

## Templates

Templates provide a way to bulk add additional fields to objects.

To do this, the template is declared for the top level Acorn object, and then a set of `[]` to bind to the nested objects field.

```acorn
args: dev: false
containers: {
    app: {}
    db: {}
}

// ... Other objects ...

if !args.dev {
    containers: [string]: {
        probes: [
            // ... probe definitions
        ]
    }

    containers: [Name= =~ "db"]: {
        ports: internal: "\(Name)-metrics-port:5000/http" // Metrics port
    }
}
```

In the above example when the `args.dev` variable is not set, all containers would have [probes](./containers#probes) assigned. In the case of the `db` container it would have a metrics port defined. The field's name is assigned to the `Name` variable if the regex matches `db`, the `Name` variable can then be referenced in the template.
