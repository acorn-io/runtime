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

Acorn includes a built-in library of functions to perform common operations underneath the `std` object:

### atoi

```cue
std.atoi(string): number
```

Parses a string into a number.

```
std.atoi("42") // 42
std.atoi("42.1") // 42.1
```

### base64, base64decode

```cue
std.base64(bytes): base64_string
std.base64decode(base64_string): bytes
```

Encodes and decods base64.

```cue
std.base64("hello") // "aGVsbG8="
std.base64decode("aGVsbG8=") // "hello"
```

### contains

```cue
std.contains(string, string): boolean
std.contains(list, value): boolean
std.contains(object, key): boolean
```

Returns whether the input contains a given value/key.

```cue
std.contains("asdf", "as") // true
std.contains("asdf", "foo") // false

std.contains(["asdf","bar"], "bar") // true
std.contains(["asdf","bar"], "baz") // false

std.contains({"x": "y", "a" :"b"}, "a") // true
std.contains({"x": "y", "a" :"b"}, "z") // false
```

### dirname, basename, fileExt

```cue
std.dirname(string): string
std.basename(string): string
std.fileExt(string): string
```

Extracts the dirname (parent path), basename (filename), and file extension of a path

```cue
std.dirname("/a/b/foo.txt") // "/a/b"
std.basename("/a/b/foo.txt") // "foo.txt"
std.fileExt("/a/b/foo.txt") // ".txt"
```

### endsWith, startsWith

```cue
std.endsWith(string, suffix): boolean
std.startsWith(string, prefix): boolean
```

Returns whether the input string starts or ends with a given string.

```cue
std.endsWith("foobar", "foo") // false
std.endsWith("foobar", "bar") // true

std.startsWith("foobar", "foo") // true
std.startsWith("foobar", "bar") // false
```

### fromHex, toHex

```cue
std.toHex(ascii_string): hex_string
std.fromHex(hex_string): ascii_string
```

Converts characters between ascii/binary and hex-encoding.

```cue
std.toHex('hi') // "6869"
std.fromHex("6869") // "hi"
```

### fromYAML, toYAML

```cue
std.fromYAML(yaml_string): object
std.toYAML(object): yaml_string
```

Parses a string of YAML and into an object and vice-versa.

```cue
x: std.fromYAML("""
	foo:
      bar:
        a: 1
        b: 2
	""")

// x now is equivalent to:
foo: bar: {
  a: 1
  b: 2
}

std.toYAML({foo: bar: "baz"}) // "foo:\n  bar:\n    baz"
```

### ifelse

```cue
std.ifelse(condition, value1, value2): value
```

Returns `value1` if `condition` is true, and `value2` otherwise.

```cue
std.ifelse(1 > 2, "yes", "no") // "no"
std.ifelse(1 <= 2, "yes", "no") // "yes"
```

### indexOf

```cue
str.indexOf(string, substring): number
str.indexOf(list, value): number
```

Returns the index where a given `value` exists in the input string or list.

```cue
std.indexOf("hello", "ll") // 2
std.indexOf("hello", "world") // -1

std.indexOf(["hello", "ll"], "hello") // 0
std.indexOf(["hello", "ll"], "world") // -1
```

### join, split

```cue
std.join(list, separator_string): string
std.split(string, separator_string): list
```

Splits a string up into a list based on a separator string, and vice-versa.

```cue
std.split("hi,bye", ",") // ["hi", "bye"]
std.join(["a", "b"], ",") // "a,b"
```

### len

```cue
std.len(string): number
std.len(list): number
```

Returns the length of the input string, or the number of items in the input list

```cue
std.len("hello") // 5
std.len([1,2,3]) // 3
```

### merge

```cue
std.merge(object1, object2): object
```

Recursively combines the values of two separate objects into one.
<!-- @TODO the rules for how fields are combined, if both are lists, etc. -->

```cue
x: {
    a: "b",
    c: "d",
    f: {
        a: "b",
        x: "y",
        l: [1, 2]
    }
}

y: {
    a: "b2",
    d: "e",
    f: {
        x: "y2",
        l: [1, 2, 3]
    }
}

z: std.merge(x, y)

// z is now:
z: {
    a: "b2",
    c: "d",
    d: "e",
    f: {
        a: "b"
        x: "y2"
        l: [1,2,3]
    }
}
```

### mod

```cue
mod(num, div): number
```

Computes the remainder of integer division (modulus)

```cue
mod(3,2) // 1
mod(14,5) // 4
```

### pathJoin

```cue
pathJoin(list): string
pathJoin(list, separator): string
```

Combines multiple segments of a path into one string.

```cue
std.pathJoin(["a", "//b", "c/"], "/") // "a/b/c"
std.pathJoin(["a", "//b", "c/"]) // "a/b/c"
std.pathJoin(["a", "//b", "c/"], "\\") // "a\\b\\c"
```

### range

```cue
std.range(end): list[numbers]
std.range(start, end): list[numbers]
std.range(start, end, increment): list[numbers]
```

Returns a list of numbers between `start` (inclusive, defaulting to 0) and `end` (exclusive) in steps of `increment` (defaulting to 1).

```cue
std.range(5) // [0, 1, 2, 3, 4]
std.range(2,5) // [2,3,4]
std.range(0, 5, 2) // [0, 2, 4]
```

### replace

```cue
std.repace(string, from, to): string
std.repace(string, from, to, limit): string
```

Replaces instances of `from` in the input string with `to`, upte `limit` times (if specified).

```cue
std.replace("hhh", "h", "b") // "bbb"
std.replace("hhhhh", "h", "b", 3) // "bbbhh"
```

### reverse

```cue
std.reverse(list): list
```

Reverses the order of items in a list.

```cue
std.reverse([2,5,4]) // [4,5,2]
```

### sha1sum, sha256sum, sha512sum

```cue
std.sha1sum(string): string
std.sha256sum(string): string
std.sha512sum(string): string
```

Returns the SHA-* hash of the input string.

```cue
std.sha1sum("hi") // "c22b5f9178342609428d6f51b2c5af4c0bde6a42"
std.sha256sum("hi") // "8f434346648f6b96df89dda901c5176b10a6d83961dd3c1ac88b59b2dc327aa4"
std.sha512sum("hi") // "150a14ed5bea6cc731cf86c41566ac427a8db48ef1b9fd626664b3bfbb99071fa4c922f33dde38719b8c8354e2b7ab9d77e0e67fc12843920a712e73d558e197"
```

### slice

```cue
std.slice(list, start): list
std.slice(list, start, end): list
```

Returns a subset of `list` starting at the `start` index, and ending at `end` (or the end of the list)

```cue
std.slice([1,2,3,4,5], 1) // [2,3,4,5]
std.slice([1,2,3,4,5], 1, 3) // [2,3,4]
```

### sort

```cue
std.sort(list): list
std.sort(list, compare_fn): list
```

Sorts the items in a list.

```
std.sort([2,5,4]) // [2,4,5]
std.sort([2, 5, 4], {x: int, y: int, less: x > y}) // [5,4,2]
```

### splitHostPort, joinHostPort

```cue
std.splitHostPort(host_port): [host, port]
std.joinHostPort(host, port): string
```

Separates a host+port string into its individual pieces, and vice-versa.

```cue
std.splitHostPort("example.com:443") // ["example.com", "443"]
std.splitHostPort("[1::1]:443") // ["1::1", "443"]
std.joinHostPort("1::1", 443) // "[1::1]:443"
```

### toLower, toTitle, toUpper

```cue
std.toLower(string): string
std.toTitle(string): string
std.toUpper(string): string
```

Converts the case of characters in the input string.

```cue
std.toLower("HELLO") // "hello"
std.toUpper("hello") // "HELLO"
std.toTitle("hello world") // "Hello World"
std.toTitle("hellO WorLD") // "HellO WorLD"
```

### trim, trimPrefix, trimSuffix

```cue
std.trim(string): string
std.trim(string, chars): string

std.trimPrefix(string): string
std.trimPrefix(string, chars): string

std.trimSuffix(string): string
std.trimSuffix(strin, charsg): string
```

Removes matching characters (whitespace by default) from the beginning, end or both sides of a string.

```cue
std.trim("  hi  ") // "hi"
std.trim("wwwHIwww", "w") // "HI"

std.trimPrefix("  asdf  ") // "asdf  "
std.trimPrefix("asdf", "as") // "df"

std.trimSuffix("  asdf  ") // "  asdf"
std.trimSuffix("asdf", "df") // "as"
```
