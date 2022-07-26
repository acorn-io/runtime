---
title: Advanced Topics
---

## Scaling Applications

### Stateless applications

In the case of scaling stateless applications the `scale` field can be defined on the container to increase the number of containers deployed. This is where consumers of the Acorn won't need persistent data or to know which instance will be deleted on scale down operation.

```cue
args: {
    // Stateless web servers to run
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

Applications that have stateful data or where an operator would care which order the containers will be removed in a scale down event should not use the `scale` field and should instead create unique instances of the container.

To accomplish this, users can leverage `for` loops in the Acornfile. Within the `for` loop block all items unique to that instance should be defined. In most cases, this will be a container and data volumes. It can contain any of the top level structs if needed.

```cue
import "list"

args: {
    // Number of instances
    replicas: 1
}

for i in list.Range(0, replicas, 1) {
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

The above example makes use of the "list" package which provides the `Range` function used in the `for` loop. The variable `i` will be an integer and placed into the container and volume names. When the application is scaled up, new containers will be deployed with their own data volumes. When the application is scaled down the highest numbered replicas will be removed first. The `0` replica will always be the first replica deployed and last removed.

When deploying stateful applications it is a reasonable assumption to bootstrap from the `0` instance and for new replicas to use that as the first point of contact to register.

## String manipulation

There are multiple ways to manipulate strings in the Acornfile.

### Simple string substitution

```cue
args: {
    userAdjective: string | *"cool"
}
...
localData: {
    config: {
        key: "this is something, \(args.userAdjective)"
    }
}
```

### Yaml Templates

If you would like to dump a section of the localData config into YAML format, you can use the YAML encoder package.

```cue
import "encoding/yaml"

args: {
    // User provided yaml
    userConfig: {...} | *{}
}

containers: {
    frontend: {
        ...
        files: {
            "/my/app/config.yaml": "secret://yaml-config/template"
        }
        ...
    }
}
secrets: {
    "yaml-config": {
        type: "template"
        data: {template: yaml.Marshal(localData.config)}
}
localData: {
    config: args.userConfig & {
        this: {
            isGoing: {
                to: "be a yaml file"
            }
        }
    }
}
```

In the above example the frontend config file will be rendered from user and Acorn data in YAML format.

### Tab writer

Another useful built in for rendering key value pairs with an optional separator is the `tabwritter` function.

If you need to create a file with content in this format:

`key=value`

```cue
import "text/tabwriter"
...
containers: {
    web: {
        ...
        files: {
            "/etc/config_file": "secret://config/template"
        }
        ...
    }
}
secrets: {
    "config": {
        type: "template"
        data: {
            template: tabwriter.Write([for key, value in localData.configData {"\(key)=\(value)"}])
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
