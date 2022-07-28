args: {
	filename: ""
	image:    ""
}

images: bar: build: {
	context:    "./bar"
	dockerfile: "./bar/\(args.filename)"
}

acorns: foo: image: args.image
