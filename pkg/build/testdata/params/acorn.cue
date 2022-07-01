args: {
	// This is a string
	str: string

	// This is a string with default
	strDefault: string | *"hi"

	// This is an int
	i: int

	// This is a int with default
	iDefault: int | *4

	// This is complex value
	complex: {
		nested: {
			val: string
		}
	}
}

containers: {
	foo: image: "\(args.strDefault)\(args.iDefault)"
}