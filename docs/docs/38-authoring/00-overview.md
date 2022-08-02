---
title: Overview
---

The primary goal of the Acornfile is to quickly and easily describe how to build, develop, and run containerized applications.

The syntax is very similar to JSON and YAML that you are already familiar with from other tools.

The resulting artifact defined by the Acornfile and produced during `acorn build` is a complete package of the container images, secrets, data, nested Acorns, etc. all in one OCI image that can be distributed through a registry.

## The primary building blocks

### Objects

In the Acornfile file the primary building block is an object. The generic syntax for any object is:

```cue
name: {
    // ...
}
```

They start with a name `field` and are wrap a collection of in `{}`. A more Acorn specific example is:

```cue
containers: {
    "my-app": {
        // ...
    }
}
```

In the above example, there is a object named containers, and another object named `my-app`.

For convenience you can collapse objects with only one field to a single `:` line until you have multiple fields for that value.

```shell
containers: app: {
    // ...
}

// ... or ...

containers: app: image: "nginx"

// ... or ...

containers: app: build: {
    context: "."
    target: "static"
}
```

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

## Fields

A `field` is a label and a value, we have seen multiple examples of `fields` in the previous examples. Here we will dive deeper.

### Field names

In an Acornfile fields can be strings with [a-zA-Z0-9_] without being wrapped in double quotes. You can use `-`, `/`, and `.` if you use double quotes around the field name.

```cue
// Valid field names
aLongField: ""
"/a/file/path": ""
"my-application-container": ""
```

### Assigning field values

Authors can assign values to fields that can be referenced elsewhere in the Acornfile. The syntax for assigning a field a value is shown below. Fields can have values of any of the supported types including objects and lists.

```cue
localData: {
    myVariable: ""
    myInteger: 5
    myBool: true
    myObject: {}
    myList: []
}
```

Once set these will be the default values.

### Strings

You can use multiline and single line strings.

Multiline strings are enclosed in triple quotes `"""`. The opening `"""` must be followed by a new line. The closing `"""` must also be on a new line. The whitespace directly preceding the closing quote must match the preceding whitespace on all other lines and is removed from these lines.

```cue
aString: "Hi!"
multiline: """
    Hello 
    World!
    """
```

### Numbers

You can use `int` and `float` types to represent numbers. The default is `int`.

```cue
int: 4
float: 4.0
```

### Additional types

In an Acornfile there can be `bool` values that are `true` and `false`.

There is also a `null` value.

### Comments

You can add comments to document the Acornfile. Comments must start with a `//` string.

```cue
// This is a comment
```

### Accessing fields

To reference a variable elsewhere in the file, you separate the key fields of the path by a `.`.

Given these variables:

```cue
localData: {
    myVariable: ""
    myInteger: 5
    myBool: true
    myObject: {
        aKey: "value"
    }
    "my-app": {
        // ...
    }
}

// Can Be accessed like

v:  localData.myVariable
i:  localData.myInteger
b:  localData.myBool
s:  localData.myObject
s0: localData.myObject.aKey 
s1: localData.myObject["aKey"]
a:  localData."my-app"
```

### Scopes

Fields referencing other fields will look at the nearest enclosing scope and work out until it hits the top level.

```cue
port: 3307
containers: app: {
    ports: localData.port // 3306
}
data: port // 3307
localData: {
    port: "3306"
    exposedServicePort: port // 3306
}
```

In the above example, `containers.app.ports` would be `3306` along with `localData.exposedServicePort`. Because of scoping, it would not be possible in the above example to set any value under localData to a value of `port`(3307) without reconfiguring the localData object.

### String substitution

Variable substitution in a string is done by wrapping the variable accessor in `\()` with the leading parenthesis escaped with a `\` like below.

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

String interpolation can happen in field in field names as well.

```cue
localData: {
    index: 3
}
containers: {
    "my-app-\(localData.index)": {
        // ...
    }
}
```

In the above example the container would have a field named `my-app-3`.

### Assigning variable to another field

Assigning a variable to a field uses the variable accessor.

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

### Operators

Multiple forms of arithmetic and boolean operators are supported.

Input:

```cue
a: 1 + 1
b: 1 / 1
c: 1 - 1
d: 1 * 5

// Bools
e: 1 > 0
f: 1 < 2
g: 1 == 1
h: 1 != 3
j: 6 == 4
```

Out:

```cue
a: 2
b: 1.0
c: 0
d: 5

// Bools
e: true
f: true
g: true
h: true
j: false
```

When performing integer `+`, `-`, and `*` with the result will be an integer. If `/` is done on integers the result will be a type `float`.

If an integer and float are part of any arithmetic operation, then the result will be of type `float`.

### Regular expression

The `=~` and `!~` operators can be used to check against regular expressions.

The `=~` operator will matc

## Function calls

The Acornfile provides built-in functions to perform common operations. All functions can be accessed from the `std` object.

An example of a function call is:

```cue
std.range(0,10)
```

## Conditionals

### If statements

Support for standard `if` statements are available in an Acornfile. If statements evaluate a boolean condition and performs actions when the condition is `true`.

```cue
localData: {
    scale: 1
}
if localData > 1 {
    // ... Do something
}
```

### If Else statments

Support for `ifelse` is available through the `std` function library. The function takes three arguments.

Arg 1: Boolean condition
Arg 2: `true` value
Arg 3: `false` value

```cue
containers: {
    app: {
        ports: publish: std.ifelse(args.dev, "3000/http", "80/http")
    }
}
```

## For Loops

The Acornfile syntax provides a for loop construct to iterate through objects and lists.

```cue
for i in std.range(0, 10) {
    containers: {
        "my-instance-\(i)": {
            // ...
        }
    }
}
```

### object field comprehensions

```cue
localData:{
    dataVols: {
        dbData: "/var/lib/mysql"
        backups: "/backups"
    }
}

for k, v in localData.config { 
    volumes: {
       "\(k)": {}
    }
    containers: {
       // ...
        dirs: {
          "\(v)": "volumes://\(k)"
        }
       // ...
    }
}
```

### List comprehension

Acornfile

```cue
localData: {
    list: ["one", "two", "three"]
}

localData: {
    multiLineContent: std.join([for i in localData.list {"Item: \(i)"}], "\n")
}
```

Renders to:

```cue
localData: {
 list: ["one", "two", "three"]
 multiLineContent: """
  Item: one
  Item: two
  Item: three
  """
}
```

## Templates

Templates provide a way to add conditional fields to existing stucts.

```cue
args: dev: false
containers: {
    app: {}
    db: {}
}

if !args.dev {
    containers: [string]: {
        probes: [
            // ... 
        ]
    }

    containers: [Name= =~ "db"]: {
        ports: internal: "5000/http" // Metrics port
    }
}
```

In a non-development scenario, all containers would have probes assigned and only the `db` container would have
