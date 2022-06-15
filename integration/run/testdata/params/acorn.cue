args: {
	deploy: {
		someInt: int
	}
}

containers: {
	foo: {
		env: {
			arg: "\(args.deploy.someInt)"
		}
		image: "nginx"
	}
}
