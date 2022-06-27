args: build: {
	filename: string
	image: string
}

images: bar: build: {
	context: "./bar"
	dockerfile: "./bar/\(args.build.filename)"
}

acorns: foo: image: string