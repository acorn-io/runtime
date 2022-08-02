---
title: Overview
---

The primary goal of the Acornfile is to quickly and easily describe how to build, develop, and run containerized applications.

The syntax is very similar to JSON and YAML that you're probably already familiar with from other tools.

The resulting artifact defined by the Acornfile and produced during `acorn build` is a complete package of your application.  It includes all the container images, secrets, data, nested Acorns, etc. in a single OCI image that can be distributed through a registry.

## The primary building blocks

### Objects

In the Acornfile file the primary building block is an object. The generic syntax for any object is:

```cue
key: {
    // ... fields ...
}
```

They start with a name `key` and are wrap a collection of fields and values in `{}`. A more Acorn specific example is:

```cue
containers: {
    "my-app": {
        // ...
    }
}
```

In the above example, there is a object called `containers`, which contains another object called `my-app`.  Keys do not need to be quoted, unless they contain a `-`.

For convenience, you can collapse objects which have only one field to a single `:` line and omit the braces.  For example these:

```cue
containers: app: image: "nginx"

containers: app: build: {
    context: "."
    target: "static"
}
```

are equivalent to:

```cue
containers: {
    foo: {
        image: "nginx"
    }
}

containers: {
    bar: {
        build: {
            context: "."
            target: "static"
        }
    }
}
```



### Lists

The other main building block is a list.

```cue
containers: {
    myapp: {
        // ...
        ports: [
            "80/http",
            "8080/http",
        ]
    }
}
```

Lists are surrounded by `[]`.  Items are separated by a comma.  The last item can include an optional trailing comma.

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

Variables allow you to store values and later reference them elsewhere in the Acornfile. The syntax for defining a variable is shown below. Values can be a string, number, boolean, list, object, or null.

```cue
localData: {
    myString: ""
    myInteger: 5
    myBool: true
    myObject: {}
    myList: []
}
```

### Strings

Strings can be a single line or multiline.  A single line string is surrounded by `"` quotes.

Multiline strings are enclosed in triple quotes `"""`. The opening `"""` must be followed by a newline. The closing `"""` must also be on it's own line. The whitespace directly preceding the closing quotles must match the preceding whitespace on all other lines and is not included not included in the value.  This allows you to indent the text to match current level without the indenting becoming part of the actual value.

```cue
singleLine: "Hi!"
multiLine: """
    Hello 
    World!
    """
# multiLine is equivalent to "Hello \nWorld!"
```

### Numbers

Numbers are integers by default unless they contain a decimal.

```cue
int: 4
float: 4.2
```

### Boolean

Booleans are `true` or `false`.

### Null

Null is `null`.

### Comments

You can add comments to document your Acornfile. Comments start with `//` and continue for the rest of the line

```cue
// This is a comment
some: "value"  // This is a comment about this line
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

Fields that reference another field will look for a value starting at the nearest enclosing scope and working upwards until reaching the top-level.

```cue
port: 3307
containers: app: {
    ports: localData.port // Evaluates to 3306
}
data: port // Evaluates to 3307
localData: {
    port: 3306
    exposedServicePort: port // Evaluates to 3306
}
```

In the above example, `containers.app.ports` would be `3306` along with `localData.exposedServicePort`. Because of scoping, it would not be possible in the above example to set any value under localData to a value of `port`(3307) without reconfiguring the localData object.

### String interpolation

Variables can be inserted into a string by wrapping their name with `\()`.  For example:

```cue
args: {
    userAdjective: "cool"
}

localData: {
    config: {
        key: "This is something \(args.userAdjective)"
    }
}

# localData.config.key is "This is something cool"
```

Interpolation can also be used for field names:

```cue
localData: {
    index: 3
}

containers: {
    "my-app-\(localData.index)": {
        // ...
    }
}

# A container named "my-app-3" is being defined
```

### Assigning a variable to another field

Assigning a variable to a field uses the variable accessor.

```cue
localData: {
    myTokenLength: 64
}

secrets: {
    "my-secret": {
        type: "token"
        params: {
            length: localData.myTokenLength // length is now 64
        }
    }
}
```

### Basic Operators

All the basic math and comparison operators you'd find in a typical programming language are supported:

| Operator | Symbol | Example | Result |
| :--------|:--------|:--------|:-------|
| Addition | `+` | `1 + 1` | `2` |
| Subtraction | `-` | `4 - 1` | `3` |
| Muliplication | `*` | `4 * 2` | `8` |
| Division | `/` | `5 / 2` | `2.5`|
| Greater than | `>` | `2 > 1` | `true` |
| Grather than or equal | `>=` | `2 >= 2` | `true` |
| Less than | `<` | `1 < 1` | `false` |
| Less than or equal | `<=` | `1 <= 1` | `true` |
| Equals | `==` | `1 == 2` | `false` |
| Does not equal | `!=` | `1 != 2` | `true` |
| Not | `!` | `!true` | `false` |
| Or | <code>\|\|</code> | <code>true \|\| false</code> | `true` |
| And | `&&` | `true && false` | `false` |

`-` can also be used to negate a value:
```cue
a: 42
b: -a // -42
```

Operations can be grouped with parenthesis and combined with `&&` and `||`:

```cue
a: 5
b: a/(1 + 1) // 2.5
c: (2+2)*4/8 // 2
d: -c * 10 // -20
e: b > c // true
f: e && b > 5 // false
```

### String Operators

Strings can be concatenated, repated, and compared:

| Operator | Symbol | Example | Result |
|:---------|:-------|:--------|:-------|
| Concatenate | `+` | `"hello " + "world"` | `"hello world"` |
| Repeat | `*` | `"hi"*5` | `"hihihihihi"` |
| Greater than | `>` | `"hi" > "bye"` | `true` |
| Greater than or equal | `>=` | `"foo" >= "bar"` | `true` |
| Less than | `<` | `"foo" < "foo` | `false` |
| Less than or equal | `<=` | `"foo" <= "foo"` | `true` |
| Equals | `==` | `"foo" == "bar"` | `false` |
| Does not equal | `!=` | `"foo" != "bar` | `true` |
| Matches regular expression | `=~` | `"hi bob" =~ "^h"` | `true` |
| Does not match regular expression | `!~` | `"hi bob" !~ "^h"` | `false` |

## Conditionals

### If statements

Support for standard `if` statements are available in an Acornfile. If conditions evaluate to a boolean, and apply their body if the condition is true.

```cue
localData: {
    scale: 1
}

if localData > 1 {
    // ... Do something
}
```

`if` statments can be added at any level or nested within each other, but there is no `else` in this format.

### If-else espressions

Ternary or "if-else" expressions are available through a built-in function which takes 3 arguments:

```std.ifelse(condition, value-if-true, value-if-false)```

The following example will publish either port 3000 or 80 depending on `args.dev`:

```cue
containers: {
    app: {
        ports: publish: std.ifelse(args.dev, "3000/http", "80/http")
    }
}
```

## For loops

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

### Object field comprehensions

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

### List comprehensions

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

## Function Library

Acorn includes a built-in library of functions to perform common operations.  See the [function libary](/reference/functions) for more information.
