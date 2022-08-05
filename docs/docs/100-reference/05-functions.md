---
title: Function Library
---

### atoi

```acorn
std.atoi(string): number
```

Parses a string into a number.

```
std.atoi("42") // 42
std.atoi("42.1") // 42.1
```

### base64, base64decode

```acorn
std.base64(bytes): base64_string
std.base64decode(base64_string): bytes
```

Encodes and decods base64.

```acorn
std.base64("hello") // "aGVsbG8="
std.base64decode("aGVsbG8=") // "hello"
```

### contains

```acorn
std.contains(string, string): boolean
std.contains(list, value): boolean
std.contains(object, key): boolean
```

Returns whether the input contains a given value/key.

```acorn
std.contains("asdf", "as") // true
std.contains("asdf", "foo") // false

std.contains(["asdf","bar"], "bar") // true
std.contains(["asdf","bar"], "baz") // false

std.contains({"x": "y", "a" :"b"}, "a") // true
std.contains({"x": "y", "a" :"b"}, "z") // false
```

### dirname, basename, fileExt

```acorn
std.dirname(string): string
std.basename(string): string
std.fileExt(string): string
```

Extracts the dirname (parent path), basename (filename), and file extension of a path

```acorn
std.dirname("/a/b/foo.txt") // "/a/b"
std.basename("/a/b/foo.txt") // "foo.txt"
std.fileExt("/a/b/foo.txt") // ".txt"
```

### endsWith, startsWith

```acorn
std.endsWith(string, suffix): boolean
std.startsWith(string, prefix): boolean
```

Returns whether the input string starts or ends with a given string.

```acorn
std.endsWith("foobar", "foo") // false
std.endsWith("foobar", "bar") // true

std.startsWith("foobar", "foo") // true
std.startsWith("foobar", "bar") // false
```

### fromHex, toHex

```acorn
std.toHex(ascii_string): hex_string
std.fromHex(hex_string): ascii_string
```

Converts characters between ascii/binary and hex-encoding.

```acorn
std.toHex('hi') // "6869"
std.fromHex("6869") // "hi"
```

### fromYAML, toYAML

```acorn
std.fromYAML(yaml_string): object
std.toYAML(object): yaml_string
```

Parses a string of YAML and into an object and vice-versa.

```acorn
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

```acorn
std.ifelse(condition, value1, value2): value
```

Returns `value1` if `condition` is true, and `value2` otherwise.

```acorn
std.ifelse(1 > 2, "yes", "no") // "no"
std.ifelse(1 <= 2, "yes", "no") // "yes"
```

### indexOf

```acorn
str.indexOf(string, substring): number
str.indexOf(list, value): number
```

Returns the index where a given `value` exists in the input string or list.

```acorn
std.indexOf("hello", "ll") // 2
std.indexOf("hello", "world") // -1

std.indexOf(["hello", "ll"], "hello") // 0
std.indexOf(["hello", "ll"], "world") // -1
```

### join, split

```acorn
std.join(list, separator_string): string
std.split(string, separator_string): list
```

Splits a string up into a list based on a separator string, and vice-versa.

```acorn
std.split("hi,bye", ",") // ["hi", "bye"]
std.join(["a", "b"], ",") // "a,b"
```

### len

```acorn
std.len(string): number
std.len(list): number
```

Returns the length of the input string, or the number of items in the input list

```acorn
std.len("hello") // 5
std.len([1,2,3]) // 3
```

### merge

```acorn
std.merge(object1, object2): object
```

Recursively combines the values of two separate objects into one.
<!-- @TODO the rules for how fields are combined, if both are lists, etc. -->

```acorn
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

```acorn
mod(num, div): number
```

Computes the remainder of integer division (modulus)

```acorn
mod(3,2) // 1
mod(14,5) // 4
```

### pathJoin

```acorn
pathJoin(list): string
pathJoin(list, separator): string
```

Combines multiple segments of a path into one string.

```acorn
std.pathJoin(["a", "//b", "c/"], "/") // "a/b/c"
std.pathJoin(["a", "//b", "c/"]) // "a/b/c"
std.pathJoin(["a", "//b", "c/"], "\\") // "a\\b\\c"
```

### range

```acorn
std.range(end): list[numbers]
std.range(start, end): list[numbers]
std.range(start, end, increment): list[numbers]
```

Returns a list of numbers between `start` (inclusive, defaulting to 0) and `end` (exclusive) in steps of `increment` (defaulting to 1).

```acorn
std.range(5) // [0, 1, 2, 3, 4]
std.range(2,5) // [2,3,4]
std.range(0, 5, 2) // [0, 2, 4]
```

### replace

```acorn
std.repace(string, from, to): string
std.repace(string, from, to, limit): string
```

Replaces instances of `from` in the input string with `to`, upte `limit` times (if specified).

```acorn
std.replace("hhh", "h", "b") // "bbb"
std.replace("hhhhh", "h", "b", 3) // "bbbhh"
```

### reverse

```acorn
std.reverse(list): list
```

Reverses the order of items in a list.

```acorn
std.reverse([2,5,4]) // [4,5,2]
```

### sha1sum, sha256sum, sha512sum

```acorn
std.sha1sum(string): string
std.sha256sum(string): string
std.sha512sum(string): string
```

Returns the SHA-* hash of the input string.

```acorn
std.sha1sum("hi") // "c22b5f9178342609428d6f51b2c5af4c0bde6a42"
std.sha256sum("hi") // "8f434346648f6b96df89dda901c5176b10a6d83961dd3c1ac88b59b2dc327aa4"
std.sha512sum("hi") // "150a14ed5bea6cc731cf86c41566ac427a8db48ef1b9fd626664b3bfbb99071fa4c922f33dde38719b8c8354e2b7ab9d77e0e67fc12843920a712e73d558e197"
```

### slice

```acorn
std.slice(list, start): list
std.slice(list, start, end): list
```

Returns a subset of `list` starting at the `start` index, and ending at `end` (or the end of the list)

```acorn
std.slice([1,2,3,4,5], 1) // [2,3,4,5]
std.slice([1,2,3,4,5], 1, 3) // [2,3,4]
```

### sort

```acorn
std.sort(list): list
std.sort(list, compare_fn): list
```

Sorts the items in a list.

```
std.sort([2,5,4]) // [2,4,5]
std.sort([2, 5, 4], {x: int, y: int, less: x > y}) // [5,4,2]
```

### splitHostPort, joinHostPort

```acorn
std.splitHostPort(host_port): [host, port]
std.joinHostPort(host, port): string
```

Separates a host+port string into its individual pieces, and vice-versa.

```acorn
std.splitHostPort("example.com:443") // ["example.com", "443"]
std.splitHostPort("[1::1]:443") // ["1::1", "443"]
std.joinHostPort("1::1", 443) // "[1::1]:443"
```

### toLower, toTitle, toUpper

```acorn
std.toLower(string): string
std.toTitle(string): string
std.toUpper(string): string
```

Converts the case of characters in the input string.

```acorn
std.toLower("HELLO") // "hello"
std.toUpper("hello") // "HELLO"
std.toTitle("hello world") // "Hello World"
std.toTitle("hellO WorLD") // "HellO WorLD"
```

### trim, trimPrefix, trimSuffix

```acorn
std.trim(string): string
std.trim(string, chars): string

std.trimPrefix(string): string
std.trimPrefix(string, chars): string

std.trimSuffix(string): string
std.trimSuffix(strin, charsg): string
```

Removes matching characters (whitespace by default) from the beginning, end or both sides of a string.

```acorn
std.trim("  hi  ") // "hi"
std.trim("wwwHIwww", "w") // "HI"

std.trimPrefix("  asdf  ") // "asdf  "
std.trimPrefix("asdf", "as") // "df"

std.trimSuffix("  asdf  ") // "  asdf"
std.trimSuffix("asdf", "df") // "as"
```
