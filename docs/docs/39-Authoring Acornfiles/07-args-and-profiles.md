---
title: Args and Profiles
---

## Args

Args are provided to allow users to provide input at different points in the Acorn lifecycle. Args allow Acornfile authors to let the user provide data and values to best suit their needs. Args should be dynamic bits of information. If they are static, use the localData structure to store the variables.

Args are defined in the top level `args` struct.

### Defining default values

Arguments to an Acorn can be standard `strings`, `ints`, `bools`, and other complex types. When defining an argument, a standard default value should be provided. The syntax to define the type and default value is:

```cue
args: {
    argName: "default"
    intVar: 1
    stringVar: "somestring"
    boolVar: true
}
```

Arg names should be in camelCase, and when entered by the user the will be dash separated.

`thisVariableName` becomes `--this-variable-name` when the user passes it on the command line.

### Provide the user some guidance

When defining arguments to the Acorn, it is helpful to the end user to also provide some context. When the user runs `acorn [IMAGE] --help` the output shows all available arguments and if defined provides a short help string.

When defining args add a `// Comment` above the argument. That will be shown the user when they do a `--help`

```cue
args: {
    // Number of instances to run.
    replicas: 1
}
```

When the user passes the `--help` arg to Acorn for this image they will see

```shell
$ acorn MYIMAGE --help
// ...
--replicas Number of instances to run. 
// ...
```

### Complex data types

Sometimes more complex data types are needed from the user. If the Acorn provides the minimum production ready configuration for an app, but some users might want to use more advanced features, authors can allow passing in `yaml` or `cue` structs from files.

Authors define the variable like:

```cue
args: {
    // User configuration data for XYZ tool
    userConfigData:  {}
}
```

The user can then create a `config.yaml` file like:

```yaml
toplevel:
  config:
  - key1: "value"
  - key2: "valueOther"
```

The config file can then be passed to the Acorn using  
`acorn run [IMAGE] --user-config-data @config.yaml`

### Built-in

To prevent the author from having to create a profile, Acorn provides the `args.dev` boolean value. It is set to `true` when running in dev mode (`acorn run -i`). Acorn authors can use this boolean with `if` statements to change dev vs. production runtime behaviors.

```cue
containers: {
    web: {
        // ...
        if args.dev {
            ports: publish: "1313/http"
        }
        if !args.dev {
            ports: publish: "80/http"
        }
    }
}
```

## Profiles

Profiles specify default arguments for different contexts like dev, test, and prod. This makes it easier for the end user to consume the Acorn application. When developing an application, often there are non-prod ports, different Dockerfile build targets, and replica counts differ from prod. Authors can define a different set of defaults for each environment.

```cue
args: {
    // Number of instances to run
    replicas: 3
}
profiles: {
    dev: {
        replicas: 1
    }
}
```

In this case when an Acorn consumer deploys the Acorn in production, 3 replicas will be deployed. When the developer working on this app runs it locally with `acorn run --profile dev .` there will only be a single replica deployed by default.

In either case, consumers of the Acorn can pass `--replicas #` to customize the deployment.

## Using args in the Acornfile

### As an environment variable or input to localData

When the value is assigned to any key in the config file, you can use '.' notation to reference the variable.

```cue
args: {
    // URL to documentation website
    docUrl: ""

    // App Config Value
    configValue: "follower"
}
containers: {
    web: {
        // ...
        env: {
            "APP_DOC_URL": args.docUrl
        }
    }
}
localData: {
    web: {
        config: {
            key: args.configValue
        }
    }
}
```

### In a string or template

When using an arg in a string or template the '.' variable needs to be placed in "\()".

```cue
args: {
    // A string arg
    aStringArg: "default"
}
// ...
secrets: {
    type: "template"
    data: {
        template: """
        a_config_line=\(args.aStringArg)
        """
    }
}
```

### Complex data input / merging

When allowing the user to pass complex structures to the Acorn, you can merge that with data predefined in localData. If you would like the user to be able to override the default localData copy of the config, you will need to also define it with `<type> | *<default>` in the localData section of the Acornfile.

Merging data is done with the `&` operator.

Here is an example of allowing the user to override some defaults, but pass in additional configuration.

```cue
args: {
    userConfig: {}
}
// ...
localData: {
    appConfig: args.userConfig & {
        userDefinableInt: 3
        staticConfigString: "this is static"
    }
}
```

In the above if the user passes a config that contains a `userDefinableInt` value the user value will be used. If the user passes `staticConfigString` in their input, Acorn will error out letting the user know that value is already defined. Everything else the user passes will be added to the `appConfig` structure.
