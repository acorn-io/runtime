args: {
	someInt: int
}

containers: {
	foo: {
		env: {
			arg: "\(args.someInt)"
		}
		image: "nginx"
	}
}
