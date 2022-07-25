---
title: Overview
---

## About the Acornfile syntax

The primary goal of the Acornfile is to quickly and easily describe how to build, develop, and run containerized applications.

The syntax is very similar to JSON and YAML that you are already familiar with from other tools.

The resulting artifact defined by the Acornfile and produced during `acorn build` is a complete bundle of the container images, secrets, data, nested acorns, etc. all packaged in a single OCI image that can be distributed through a registry.

### The primary building blocks

In the Acornfile file the primary building block is a struct. The generic syntax for any struct is:

```cue
name: {
    ...
}
```

A more Acorn specific example is:

```cue
containers: {
    "my-app": {
        ...
    }
}
```

In the above example, there is a struct called containers, and another stuct called `my-app`. It should be noted in an Acornfile, user-defined keys like `my-app` need to be placed in quotes if they contain a `-`.

### Lists

The other main building block type are lists.

```cue
containers: {
    myapp: {
        ...
        ports: [
            "80/http",
        ]
        ...
    }
}
```

The list is surrounded by `[]` and each item has a trailing comma.

### Variables

You can also set variables, where you can define both the type and a default if desired.

```cue
localData: {
    myVariable: ""
    myInteger: 5
    myBool: true
    myStruct: {}
}
```

Each of the above variables is set as a default value.

Once a variable is set, it can not be changed.

To reference a variable elsewhere in the file, you separate the key fields by a `.`. When using the value in a string, you need to wrap it in `()` with the leading parenthesis escaped with a `\` like below.

```cue
localData: {
    myVariable: "hello world!"
}

containers: {
    web: {
        ...
        env: {
            "hi! \(localData.myVariable)"
        }
    }
}
```

Assigning a variable to another value looks like:

```cue
localData: {
    myTokenLength: 64
}

secrets: {
    "my-secret": {
        type: "token"
        params: {
            length: localData.myTokenLength
        }
    }
}
```
