params: build: {
	filename: string
}

images: bar: build: {
	context: "./bar"
	dockerfile: "./bar/\(params.build.filename)"
}

// busybox is not a proper acorn image, but should work since we don't validate and we know it's an index
acorns: foo: image: "busybox"