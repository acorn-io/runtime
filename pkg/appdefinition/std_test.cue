containers: default: {
	image: "nginx"
	files: "a": std.toYAML(localData)
}

localData: {
	t: true

	range: std.range(5)
	range: std.range(0, 5)
	range: [0, 1, 2, 3, 4]

	range2: std.range(0, 5, 2)
	range2: [0, 2, 4]

	ifelse: std.ifelse(range2[0] == 1, "is one", "is not one")
	ifelse: std.ifelse(range2[0] == 0, "is not one", "is one")
	ifelse: "is not one"

	fromYAML: std.fromYAML("""
		foo: bar
		""")
	fromYAML: foo: "bar"

	sha1sum: std.sha1sum("hi")
	sha1sum: "c22b5f9178342609428d6f51b2c5af4c0bde6a42"

	sha256sum: std.sha256sum("hi")
	sha256sum: "8f434346648f6b96df89dda901c5176b10a6d83961dd3c1ac88b59b2dc327aa4"

	sha512sum: std.sha512sum("hi")
	sha512sum: "150a14ed5bea6cc731cf86c41566ac427a8db48ef1b9fd626664b3bfbb99071fa4c922f33dde38719b8c8354e2b7ab9d77e0e67fc12843920a712e73d558e197"

	base64: std.base64("hello")
	base64: "aGVsbG8="

	base64decode: std.base64decode("aGVsbG8=")
	base64decode: 'hello'

	toHex: std.toHex("hi")
	toHex: std.toHex('hi')
	toHex: "6869"

	fromHex: std.fromHex("6869")
	fromHex: 'hi'

	toJSON: std.toJSON({foo: "bar"})
	toJSON: "{\"foo\":\"bar\"}"

	fromJSON: std.fromJSON("""
		{"foo":"bar"}
		""")
	fromJSON: {foo: "bar"}

	slice: std.slice([1, 2, 3], 1, 2)
	slice: [2]

	sort: std.sort([2, 5, 4], {x: int, y: int, less: x > y})
	sort: std.reverse([2, 4, 5])
	sort: [5, 4, 2]
	sort2: std.sort([2, 5, 4])
	sort2: [2, 4, 5]

	splitHostPort: std.splitHostPort("example.com:443")
	splitHostPort: ["example.com", "443"]

	splitHostPort2: std.splitHostPort("[1::1]:443")
	splitHostPort2: ["1::1", "443"]

	joinHostPort: std.joinHostPort("1::1", 443)
	joinHostPort: "[1::1]:443"

	pathJoin: std.pathJoin(["a", "//b", "c/"], "/")
	pathJoin: "a/b/c"
	pathJoin: std.pathJoin(["a", "//b", "c/"])
	pathJoin: "a/b/c"

	pathJoin2: std.pathJoin(["a", "//b", "c/"], "\\")
	pathJoin2: "a\\b\\c"

	dirname: std.dirname("a/b")
	dirname: "a"

	basename: std.basename("a/b")
	basename: "b"

	fileExt: std.fileExt("cmd.bat")
	fileExt: ".bat"

	atoi: std.atoi("4")
	atoi: 4

	anum: 4
	itoa: "\(anum)"
	itoa: "4"

	toTitle: std.toTitle("hello")
	toTitle: "Hello"

	contains: true
	contains: std.contains("asdf", "as")
	contains: std.contains(["asdf","bar"], "bar")
	contains: std.contains({"x": "y", "a" :"b"}, "a")

	split: std.split("hi,bye", ",")
	split: ["hi", "bye"]

	split2: std.split("hi,bye,foo", ",", 2)
	split2: ["hi", "bye,foo"]

	join: std.join(["a", "b"], ",")
	join: "a,b"

	endsWith: std.endsWith("foobar", "foo")
	endsWith: false

	startsWith: std.startsWith("foobar", "foo")
	startsWith: true

	toUpper: std.toUpper("hi")
	toUpper: "HI"

	toLower: std.toLower("HI")
	toLower: "hi"

	trim: std.trim("  hi  ")
	trim: "hi"

	trimSuffix: std.trimSuffix("asdf", "df")
	trimSuffix: "as"

	trimPrefix: std.trimPrefix("asdf", "as")
	trimPrefix: "df"

	replace: std.replace("hhh", "h", "b")
	replace: "bbb"

	replace2: std.replace("hhhhh", "h", "b", 3)
	replace2: "bbbhh"

	indexOf: std.indexOf("hello", "ll")
	indexOf: 2

	indexOf2: std.indexOf(["hello", "ll"], "ll")
	indexOf2: 1

	merge: std.merge({"a": "b", "c": "d", f: {"a": "b", "x": "y", "l": [1, 2]}}, {"a": "b1", "d": "e", f: {"x": "y1", "l": [1, 2, 3]}})
	t:     merge.a == "b1"
	t:     merge.c == "d"
	t:     merge.d == "e"
	t:     merge.f.x == "y1"
	t:     merge.f.a == "b"
	t:     merge.f.l[2] == 3
}
