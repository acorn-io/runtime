args: {
	filename: string
	image:    string
}

images: bar: build: {
	context:    "./bar"
	dockerfile: "./bar/\(args.filename)"
}

acorns: foo: image: string
