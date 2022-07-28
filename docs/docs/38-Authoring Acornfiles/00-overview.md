---
title: Overview
---

## About the Acornfile syntax

The primary goal of the Acornfile is to quickly and easily describe how to build, develop, and run containerized applications.

The syntax is very similar to JSON and YAML that you are already familiar with from other tools.

The resulting artifact defined by the Acornfile and produced during `acorn build` is a complete package of the container images, secrets, data, nested Acorns, etc. all in one OCI image that can be distributed through a registry.

## The primary building blocks

### Structs

In the Acornfile file the primary building block is a struct. The generic syntax for any struct is:

```cue
name: {
    // ...
}
```

A more Acorn specific example is:

```cue
containers: {
    "my-app": {
        // ...
    }
}
```

In the above example, there is a struct called containers, and another struct called `my-app`. It should be noted in an Acornfile, user-defined keys like `my-app` need to be placed in quotes if they contain a `-`.

### Lists

The other main building block type are lists.

```cue
containers: {
    myapp: {
        // ...
        ports: [
            "80/http",
        ]
        // ...
    }
}
```

The list is surrounded by `[]` and each item has a trailing comma, including the last item.

## Variables

Variables allow the author to assign values to names that can be referenced elsewhere in the Acornfile. The syntax for defining a variable is shown below.

```cue
localData: {
    myVariable: ""
    myInteger: 5
    myBool: true
    myStruct: {}
}
```

Each of the above variables is set as a default value.

### Variable string substitution

To reference a variable elsewhere in the file, you separate the key fields of the path by a `.`. When using the value in a string, you need to wrap it in `\()` with the leading parenthesis escaped with a `\` like below.

```cue
args: {
    userAdjective: "cool"
}
// ...
localData: {
    config: {
        key: "this is something, \(args.userAdjective)"
    }
}
```

### Assigning variable to another field

Assigning a variable to a field uses the `.` notation to reference the variable.

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

### Comments

You can add comments to document the Acornfile. Comments must start with a `//` string.

```cue
// This is a comment
```
